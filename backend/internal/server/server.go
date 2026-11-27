package server

import (
	"context"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"maps"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type account struct {
	PINHash string `json:"pin_hash,omitempty"`
	Salt    string `json:"salt"`
	Hash    string `json:"hash"`
}
type loginAttempts struct {
	Count int   `json:"count"`
	Until int64 `json:"until"`
}
type state struct {
	LibraryFolder    string                         `json:"library_folder,omitempty"`
	Devices          map[string]DevicePairing       `json:"devices,omitempty"`
	Usage            map[string]DailyUsage          `json:"usage,omitempty"`
	Bookmarks        map[string]map[string]Bookmark `json:"bookmarks,omitempty"`
	ApprovalLog      []ApprovalEvent                `json:"approval_log,omitempty"`
	ArchivedChannels json.RawMessage                `json:"archived_channels,omitempty"`
	Viewing          map[string]Viewing             `json:"viewing,omitempty"`
	Media            []Media                        `json:"media,omitempty"`
	Attempts         map[string]loginAttempts       `json:"login_attempts,omitempty"`
	Account          *account                       `json:"account,omitempty"`
	Sessions         map[string]int64               `json:"sessions,omitempty"`
	Profiles         []ChildProfile                 `json:"profiles,omitempty"`
}
type Server struct {
	scanner       *LibraryScanner
	currentState  func(context.Context) (state, error)
	mu            sync.Mutex
	path          string
	data          state
	secureCookies bool
	save          func(state) error
}

func New(path string, secure bool) (*Server, error) {
	s := &Server{path: path, secureCookies: secure}
	b, e := os.ReadFile(path)
	if e == nil {
		if b, e = retireChannels(b); e != nil {
			return nil, e
		}
		if e = json.Unmarshal(b, &s.data); e != nil {
			return nil, e
		}
	} else if !errors.Is(e, os.ErrNotExist) {
		return nil, e
	}
	if s.data.Sessions == nil {
		s.data.Sessions = map[string]int64{}
	}
	return s, nil
}
func (s *Server) persist() error {
	if s.save != nil {
		return s.save(s.data)
	}
	if e := os.MkdirAll(filepath.Dir(s.path), 0700); e != nil {
		return e
	}
	b, e := json.Marshal(s.data)
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(s.path), ".familytv-*")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	if e = f.Chmod(0600); e != nil {
		f.Close()
		return e
	}
	if _, e = f.Write(b); e != nil {
		f.Close()
		return e
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	return os.Rename(f.Name(), s.path)
}
func token() string {
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
func digest(pass, salt string) string {
	key, err := pbkdf2.Key(sha256.New, pass, []byte(salt), 310000, 32)
	if err != nil {
		panic(err)
	} // Fixed parameters are valid.
	return base64.RawStdEncoding.EncodeToString(key)
}
func write(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}

type parentCredentials struct {
	Password string `json:"password"`
	PIN      string `json:"pin"`
}

func validPIN(pin string) bool {
	return pin == "" || (len(pin) >= 6 && len(pin) <= 12 && strings.Trim(pin, "0123456789") == "")
}
func credentials(r *http.Request) (parentCredentials, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, 4096)
	var x parentCredentials
	if e := json.NewDecoder(r.Body).Decode(&x); e != nil {
		return x, e
	}
	if len(x.Password) < 12 || len(x.Password) > 256 || !validPIN(x.PIN) {
		return x, errors.New("invalid credentials")
	}
	return x, nil
}
func (s *Server) authorized(r *http.Request) bool {
	c, e := r.Cookie("familytv_admin")
	if e != nil {
		return false
	}
	expiry, ok := s.data.Sessions[c.Value]
	return ok && expiry > time.Now().Unix()
}
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	for _, method := range []string{"GET", "POST", "PATCH", "PUT", "DELETE"} {
		mux.HandleFunc(method+" /api/", http.NotFound)
	}
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		write(w, 200, map[string]string{"status": "ok", "service": "FamilyTV"})
	})
	mux.HandleFunc("GET /api/setup/status", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		write(w, 200, map[string]bool{"needs_setup": s.data.Account == nil})
	})
	mux.HandleFunc("POST /api/setup", func(w http.ResponseWriter, r *http.Request) {
		pass, e := credentials(r)
		if e != nil {
			write(w, 400, map[string]string{"error": "invalid password"})
			return
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.data.Account != nil {
			write(w, 409, map[string]string{"error": "already configured"})
			return
		}
		salt := token()
		s.data.Account = &account{Salt: salt, Hash: digest(pass.Password, salt)}
		if pass.PIN != "" {
			s.data.Account.PINHash = digest(pass.PIN, "pin:"+salt)
		}
		if e = s.persist(); e != nil {
			s.data.Account = nil
			write(w, 500, map[string]string{"error": "storage failure"})
			return
		}
		write(w, 201, map[string]bool{"created": true})
	})
	mux.HandleFunc("POST /api/login", func(w http.ResponseWriter, r *http.Request) {
		pass, e := credentials(r)
		if e != nil {
			write(w, 401, map[string]string{"error": "invalid credentials"})
			return
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		now := time.Now()
		if s.data.Account == nil {
			write(w, 409, map[string]string{"error": "setup required"})
			return
		}
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}
		// Scope the key to this installation and keep client addresses out of saved state.
		sum := sha256.Sum256([]byte(s.data.Account.Salt + " " + host))
		client := base64.RawURLEncoding.EncodeToString(sum[:])
		attempts := maps.Clone(s.data.Attempts)
		if attempts == nil {
			attempts = make(map[string]loginAttempts)
		}
		for key, entry := range attempts {
			if entry.Until <= now.Unix() {
				delete(attempts, key)
			}
		}
		attempt := attempts[client]
		if attempt.Count >= 5 || (attempt.Count == 0 && len(attempts) >= 1024) {
			write(w, 429, map[string]string{"error": "too many attempts"})
			return
		}
		hash := digest(pass.Password, s.data.Account.Salt)
		pinOK := true
		if s.data.Account.PINHash != "" {
			pinOK = subtle.ConstantTimeCompare([]byte(digest(pass.PIN, "pin:"+s.data.Account.Salt)), []byte(s.data.Account.PINHash)) == 1
		}
		before := s.data
		if subtle.ConstantTimeCompare([]byte(hash), []byte(s.data.Account.Hash)) != 1 || !pinOK {
			attempt.Count++
			if attempt.Count == 1 || attempt.Count == 5 {
				attempt.Until = now.Add(5 * time.Minute).Unix()
			}
			attempts[client] = attempt
			s.data.Attempts = attempts
			if err := s.persist(); err != nil {
				s.data = before
				write(w, 500, map[string]string{"error": "storage failure"})
				return
			}
			write(w, 401, map[string]string{"error": "invalid credentials"})
			return
		}
		delete(attempts, client)
		s.data.Attempts = attempts
		s.data.Sessions = maps.Clone(s.data.Sessions)
		for key, expiry := range s.data.Sessions {
			if expiry <= now.Unix() {
				delete(s.data.Sessions, key)
			}
		}
		t := token()
		s.data.Sessions[t] = now.Add(12 * time.Hour).Unix()
		if err := s.persist(); err != nil {
			s.data = before
			write(w, 500, map[string]string{"error": "storage failure"})
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "familytv_admin", Value: t, Path: "/", HttpOnly: true, Secure: s.secureCookies, SameSite: http.SameSiteStrictMode, MaxAge: 43200})
		write(w, 200, map[string]bool{"authenticated": true})
	})
	mux.HandleFunc("PUT /api/admin/pin", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		if !s.authorized(r) {
			write(w, 401, map[string]string{"error": "Sign in again."})
			return
		}
		var input struct {
			PIN *string `json:"pin"`
		}
		if !readInput(w, r, &input) {
			return
		}
		if input.PIN == nil || !validPIN(*input.PIN) {
			write(w, 400, map[string]string{"error": "Enter 6 to 12 digits."})
			return
		}
		before := s.data
		updated := *s.data.Account
		updated.PINHash = ""
		if *input.PIN != "" {
			updated.PINHash = digest(*input.PIN, "pin:"+updated.Salt)
		}
		s.data.Account = &updated
		cookie, _ := r.Cookie("familytv_admin")
		s.data.Sessions = map[string]int64{cookie.Value: before.Sessions[cookie.Value]}
		if err := s.persist(); err != nil {
			s.data = before
			write(w, 500, map[string]string{"error": "Storage failure."})
			return
		}
		write(w, 200, map[string]bool{"saved": true})
	})
	mux.HandleFunc("POST /api/logout", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		if c, e := r.Cookie("familytv_admin"); e == nil && s.data.Sessions[c.Value] != 0 {
			expiry := s.data.Sessions[c.Value]
			delete(s.data.Sessions, c.Value)
			if e = s.persist(); e != nil {
				s.data.Sessions[c.Value] = expiry
				write(w, 500, map[string]string{"error": "storage failure"})
				return
			}
		}
		http.SetCookie(w, &http.Cookie{Name: "familytv_admin", Path: "/", MaxAge: -1, HttpOnly: true, Secure: s.secureCookies, SameSite: http.SameSiteStrictMode})
		write(w, 200, map[string]bool{"authenticated": false})
	})
	for _, method := range []string{"GET", "POST", "PATCH", "DELETE"} {
		mux.HandleFunc(method+" /api/profiles/{id}/session", s.parentViewing)
	}
	mux.HandleFunc("POST /api/profiles/{id}/session/pair", s.parentViewing)
	mux.HandleFunc("PUT /api/profiles/{id}/session/pair", s.parentViewing)
	mux.HandleFunc("DELETE /api/profiles/{id}/session/pair", s.parentViewing)
	for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE"} {
		mux.HandleFunc(method+" /api/devices", s.accountDevices)
	}
	mux.HandleFunc("POST /api/watch/device/start", s.deviceStart)
	mux.HandleFunc("GET /api/watch/device", s.deviceStatus)
	mux.HandleFunc("POST /api/watch/join", s.joinViewing)
	mux.HandleFunc("GET /api/watch", s.watchCatalog)
	mux.HandleFunc("GET /api/watch/ending-music", s.endingMusic)
	mux.HandleFunc("POST /api/watch/bookmark", s.bookmark)
	mux.HandleFunc("POST /api/watch/leave", s.watchAction)
	mux.HandleFunc("GET /api/watch/media/{id}/compatible", s.viewerMediaFile)
	mux.HandleFunc("POST /api/watch/play", s.watchAction)
	mux.HandleFunc("POST /api/watch/finish", s.watchAction)
	mux.HandleFunc("GET /api/watch/media/{id}/picture", s.picture)
	mux.HandleFunc("GET /api/watch/media/{id}/file", s.viewerMediaFile)
	mux.HandleFunc("GET /api/watch/media/{id}/captions", s.viewerMediaFile)
	mux.HandleFunc("GET /api/metadata/lookup", s.metadataLookup)
	mux.HandleFunc("POST /api/media/{id}/probe", s.refreshProbe)
	mux.HandleFunc("POST /api/media/library-scan", s.libraryScan)
	mux.HandleFunc("GET /api/media/library-scan", s.libraryScan)
	mux.HandleFunc("GET /api/media/library-folder", s.libraryFolder)
	mux.HandleFunc("PUT /api/media/library-folder", s.libraryFolder)
	mux.HandleFunc("GET /api/media/scan", s.scanMedia)
	mux.HandleFunc("PUT /api/media/{id}/metadata", s.editMetadata)
	mux.HandleFunc("DELETE /api/media/{id}/metadata", s.editMetadata)
	mux.HandleFunc("GET /api/approvals", s.approvals)
	mux.HandleFunc("POST /api/approvals", s.approvals)
	mux.HandleFunc("GET /api/coverage/{id}", s.coverage)
	mux.HandleFunc("GET /api/media", s.mediaList)
	mux.HandleFunc("POST /api/media", s.importMedia)
	mux.HandleFunc("PUT /api/media/{id}/review", s.reviewMedia)
	mux.HandleFunc("DELETE /api/media/{id}", s.removeMedia)
	mux.HandleFunc("GET /api/media/{id}/file", s.parentMediaFile)
	mux.HandleFunc("GET /api/media/{id}/captions", s.parentMediaFile)
	mux.HandleFunc("GET /api/profiles", s.listProfiles)
	mux.HandleFunc("POST /api/profiles", s.createProfile)
	mux.HandleFunc("DELETE /api/profiles/{id}", s.deleteProfile)
	mux.HandleFunc("PUT /api/profiles/{id}", s.updateProfile)
	mux.HandleFunc("GET /api/profiles/export", s.exportProfiles)
	mux.HandleFunc("GET /api/admin", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		if !s.authorized(r) {
			write(w, 401, map[string]string{"error": "authentication required"})
			return
		}
		write(w, 200, map[string]string{"service": "FamilyTV", "stage": "M0"})
	})
	if dir := os.Getenv("FAMILYTV_WEB_DIR"); dir != "" {
		fs := http.FileServer(http.Dir(dir))
		mux.Handle("GET /", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.NotFound(w, r)
				return
			}
			path := filepath.Join(dir, filepath.Clean(r.URL.Path))
			if _, e := os.Stat(path); e != nil {
				r2 := r.Clone(r.Context())
				r2.URL.Path = "/"
				fs.ServeHTTP(w, r2)
				return
			}
			fs.ServeHTTP(w, r)
		}))
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		if r.Method != "GET" && r.Method != "HEAD" && r.Method != "OPTIONS" && r.Header.Get("Origin") != "" {
			u, e := url.Parse(r.Header.Get("Origin"))
			if e != nil || u.Host != r.Host || (u.Scheme != "http" && u.Scheme != "https") || (s.secureCookies && u.Scheme != "https") {
				http.Error(w, "forbidden origin", 403)
				return
			}
		}
		mux.ServeHTTP(w, r)
	})
}
