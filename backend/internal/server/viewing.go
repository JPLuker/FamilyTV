package server

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"strings"
	"time"
)

type Viewing struct {
	Channel    string `json:"channel,omitempty"`
	Profile    string `json:"profile"`
	Mode       string `json:"mode"`
	Deadline   int64  `json:"deadline,omitempty"`
	Current    string `json:"current,omitempty"`
	Last       bool   `json:"last"`
	JoinHash   string `json:"join_hash,omitempty"`
	JoinUntil  int64  `json:"join_until,omitempty"`
	CookieHash string `json:"cookie_hash,omitempty"`
}
type WatchItem struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Subject  string  `json:"subject"`
	Duration float64 `json:"duration_seconds"`
	Captions bool    `json:"captions"`
}

func secretHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
func readInput(w http.ResponseWriter, r *http.Request, value any) bool {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192))
	d.DisallowUnknownFields()
	if d.Decode(value) != nil || d.Decode(&struct{}{}) != io.EOF {
		write(w, 400, map[string]string{"error": "Invalid request."})
		return false
	}
	return true
}
func (s *Server) profile(id string) (ChildProfile, bool) {
	for _, p := range s.data.Profiles {
		if p.ID == id {
			return p, true
		}
	}
	return ChildProfile{}, false
}
func (s *Server) setViewing(w http.ResponseWriter, id string, next *Viewing) bool {
	if s.data.Viewing == nil {
		s.data.Viewing = map[string]Viewing{}
	}
	old, had := s.data.Viewing[id]
	if next == nil {
		delete(s.data.Viewing, id)
	} else {
		s.data.Viewing[id] = *next
	}
	if err := s.persist(); err != nil {
		if had {
			s.data.Viewing[id] = old
		} else {
			delete(s.data.Viewing, id)
		}
		write(w, 500, map[string]string{"error": "storage failure"})
		return false
	}
	return true
}
func active(v Viewing, now time.Time) bool {
	return v.Profile != "" && (v.Deadline == 0 || v.Deadline > now.UnixMilli())
}
func (s *Server) parentViewing(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.authorized(r) {
		write(w, 401, map[string]string{"error": "authentication required"})
		return
	}
	id := r.PathValue("id")
	if _, ok := s.profile(id); !ok {
		http.NotFound(w, r)
		return
	}
	now := time.Now()
	v := s.data.Viewing[id]
	if strings.HasSuffix(r.URL.Path, "/pair") {
		if !active(v, now) {
			write(w, 409, map[string]string{"error": "Start a viewing session first."})
			return
		}
		s.issueCode(w, id, v, now)
		return
	}
	if r.Method == "GET" {
		write(w, 200, map[string]any{"active": active(v, now), "mode": v.Mode, "deadline": v.Deadline, "playing": v.Current != "" || v.Channel != "", "last": v.Last})
		return
	}
	if r.Method == "DELETE" {
		if s.setViewing(w, id, nil) {
			write(w, 200, map[string]bool{"ended": true})
		}
		return
	}
	if r.Method == "PATCH" {
		if v.Channel != "" {
			p, _ := s.profile(id)
			g := s.currentGuide(p, v, now.UnixMilli())
			if g != nil && g.Now != nil {
				v.Last = true
				v.Deadline = g.Now.End
				if s.setViewing(w, id, &v) {
					write(w, 200, map[string]bool{"last": true})
				}
				return
			}
		}
		if !active(v, now) || v.Current == "" {
			if s.setViewing(w, id, nil) {
				write(w, 200, map[string]bool{"ended": true})
			}
			return
		}
		v.Last = true
		if s.setViewing(w, id, &v) {
			write(w, 200, map[string]bool{"last": true})
		}
		return
	}
	var input struct {
		Mode     string `json:"mode"`
		Minutes  int    `json:"minutes"`
		FinishBy string `json:"finish_by"`
	}
	if !readInput(w, r, &input) {
		return
	}
	if active(v, now) {
		write(w, 409, map[string]string{"error": "End the active session first."})
		return
	}
	v = Viewing{Profile: id, Mode: input.Mode}
	switch input.Mode {
	case "manual":
	case "one_video":
		v.Last = true
	case "duration":
		if input.Minutes <= 0 || input.Minutes > 1440 {
			write(w, 400, map[string]string{"error": "Choose 1 to 1440 minutes."})
			return
		}
		v.Deadline = now.Add(time.Duration(input.Minutes) * time.Minute).UnixMilli()
	case "finish_by":
		t, e := time.Parse(time.RFC3339, input.FinishBy)
		if e != nil || !t.After(now) {
			write(w, 400, map[string]string{"error": "Choose a future time with a time zone."})
			return
		}
		v.Deadline = t.UnixMilli()
	default:
		write(w, 400, map[string]string{"error": "Unknown session mode."})
		return
	}
	s.issueCode(w, id, v, now)
}
func (s *Server) issueCode(w http.ResponseWriter, id string, v Viewing, now time.Time) {
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		write(w, 500, map[string]string{"error": "Pairing unavailable."})
		return
	}
	code := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw[:])
	v.JoinHash = secretHash(code)
	v.JoinUntil = now.Add(10 * time.Minute).UnixMilli()
	if s.setViewing(w, id, &v) {
		write(w, 201, map[string]any{"code": code, "deadline": v.Deadline})
	}
}
func (s *Server) joinViewing(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Code string `json:"code"`
	}
	if !readInput(w, r, &input) {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	hash := secretHash(input.Code)
	normalized := secretHash(strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(input.Code), "-", "")))
	for id, v := range s.data.Viewing {
		if (v.JoinHash != hash && v.JoinHash != normalized) || !active(v, now) || v.JoinUntil <= now.UnixMilli() {
			continue
		}
		if _, ok := s.profile(id); !ok {
			continue
		}
		cookie := token()
		v.CookieHash = secretHash(cookie)
		v.JoinHash = ""
		v.JoinUntil = 0
		if s.setViewing(w, id, &v) {
			http.SetCookie(w, &http.Cookie{Name: "familytv_viewer", Value: cookie, Path: "/api/watch", HttpOnly: true, Secure: s.secureCookies, SameSite: http.SameSiteStrictMode})
			write(w, 200, map[string]bool{"joined": true})
		}
		return
	}
	write(w, 403, map[string]string{"error": "This viewing link has expired or has already been used."})
}
func (s *Server) viewer(r *http.Request) (Viewing, ChildProfile, bool) {
	c, e := r.Cookie("familytv_viewer")
	if e != nil {
		return Viewing{}, ChildProfile{}, false
	}
	hash := secretHash(c.Value)
	for _, v := range s.data.Viewing {
		if v.CookieHash == hash && active(v, time.Now()) {
			p, ok := s.profile(v.Profile)
			return v, p, ok
		}
	}
	return Viewing{}, ChildProfile{}, false
}
func fits(m Media, v Viewing, now time.Time) bool {
	return v.Deadline == 0 || int64(math.Ceil(m.Duration*1000)) <= v.Deadline-now.UnixMilli()
}
func available(m Media, p ChildProfile) bool {
	if !eligible(m, p) {
		return false
	}
	f, info, err := openMedia(m.Path)
	if err != nil {
		return false
	}
	f.Close()
	if info.ModTime().UnixNano() != m.Modified || info.Size() != m.Size {
		return false
	}
	if p.Captions {
		f, _, err := openCaption(m.Path)
		if err != nil {
			return false
		}
		f.Close()
	}
	return true
}
func (s *Server) watchCatalog(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, p, ok := s.viewer(r)
	if !ok {
		write(w, 403, map[string]string{"error": "Session ended."})
		return
	}
	items := []WatchItem{}
	var current *WatchItem
	term := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	now := time.Now()
	for _, m := range s.data.Media {
		if !available(m, p) {
			continue
		}
		item := WatchItem{m.ID, m.Title, m.Review.Subject, m.Duration, m.Captions}
		if m.ID == v.Current {
			current = &item
		}
		if fits(m, v, now) && strings.Contains(strings.ToLower(m.Title+" "+m.Review.Subject), term) {
			items = append(items, item)
		}
	}
	guides := []Guide{}
	var live *Guide
	for _, channel := range s.data.Channels {
		g := s.guide(channel, p, v, now.UnixMilli())
		if g.Now != nil || g.Next != nil {
			guides = append(guides, g)
		}
		if channel.ID == v.Channel {
			live = &g
			if g.Now != nil {
				current = &g.Now.Item
			}
		}
	}
	write(w, 200, map[string]any{"items": items, "current": current, "channels": guides, "live": live, "tuned": v.Channel != "", "deadline": v.Deadline, "server_time": now.UnixMilli(), "captions": p.Captions, "navigation": p.Navigation, "reduced_motion": p.ReducedMotion})
}
func (s *Server) watchAction(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, p, ok := s.viewer(r)
	if !ok {
		write(w, 403, map[string]string{"error": "Session ended."})
		return
	}
	var input struct {
		ID string `json:"id"`
	}
	if !readInput(w, r, &input) {
		return
	}
	if r.URL.Path == "/api/watch/finish" {
		if v.Channel == "" && (v.Current == "" || v.Current != input.ID) {
			write(w, 409, map[string]string{"error": "Video is no longer playing."})
			return
		}
		v.Current = ""
		v.Channel = ""
		var next *Viewing = &v
		if v.Last {
			next = nil
		}
		if s.setViewing(w, p.ID, next) {
			write(w, 200, map[string]bool{"ended": next == nil})
		}
		return
	}
	if v.Channel != "" {
		write(w, 409, map[string]string{"error": "Leave the channel first."})
		return
	}
	if v.Current != "" && v.Current != input.ID {
		stillPlayable := false
		for _, m := range s.data.Media {
			if m.ID == v.Current && available(m, p) {
				stillPlayable = true
				break
			}
		}
		if stillPlayable {
			write(w, 409, map[string]string{"error": "Finish the current video first."})
			return
		}
		if v.Last {
			if s.setViewing(w, p.ID, nil) {
				write(w, 403, map[string]string{"error": "Session ended."})
			}
			return
		}
	}
	for _, m := range s.data.Media {
		if m.ID == input.ID && available(m, p) && (v.Current == m.ID || fits(m, v, time.Now())) {
			v.Current = m.ID
			if s.setViewing(w, p.ID, &v) {
				write(w, 200, map[string]bool{"playing": true})
			}
			return
		}
	}
	write(w, 404, map[string]string{"error": "Video unavailable in this session."})
}
