package server

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"maps"
	"net/http"
	"slices"
	"time"
)

// The empty key identifies the single household. Old profile keys may still
// contain a pending one-time code from before account-level pairing.
const householdDevices = ""

func (st *state) UnmarshalJSON(b []byte) error {
	type stored state
	var next stored
	if err := json.Unmarshal(b, &next); err != nil {
		return err
	}
	*st = state(next)
	for id, old := range st.Devices {
		if id == householdDevices {
			continue
		}
		account := st.Devices[householdDevices]
		for _, screen := range old.Screens {
			screen.Profile = id
			account.Screens = append(account.Screens, screen)
		}
		st.Devices[householdDevices] = account
		old.Screens = nil
		if old.Challenge == "" {
			delete(st.Devices, id)
		} else {
			st.Devices[id] = old
		}
	}
	return nil
}

type DevicePairing struct {
	AllowStart bool           `json:"allow_start,omitempty"`
	Challenge  string         `json:"challenge,omitempty"`
	Until      int64          `json:"until,omitempty"`
	Screens    []pairedScreen `json:"screens,omitempty"`
}
type pairedScreen struct {
	AllowStart bool   `json:"allow_start,omitempty"`
	Profile    string `json:"profile,omitempty"`
	Hash       string `json:"hash"`
	Until      int64  `json:"until"`
}

func newPairingCode(w http.ResponseWriter) (string, bool) {
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		write(w, 500, map[string]string{"error": "Pairing unavailable."})
		return "", false
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw[:])[:6], true
}

func (s *Server) saveDevices(w http.ResponseWriter, id string, next DevicePairing) bool {
	if s.data.Devices == nil {
		s.data.Devices = map[string]DevicePairing{}
	}
	old, had := s.data.Devices[id]
	s.data.Devices[id] = next
	if err := s.persist(); err != nil {
		if had {
			s.data.Devices[id] = old
		} else {
			delete(s.data.Devices, id)
		}
		write(w, 500, map[string]string{"error": "storage failure"})
		return false
	}
	return true
}
func (s *Server) pendingDeviceChallenge(hash string, now time.Time) bool {
	for _, p := range s.data.Devices {
		if p.Challenge == hash && p.Until > now.Unix() {
			return true
		}
	}
	return false
}
func (s *Server) accountDevices(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.authorized(r) {
		write(w, 401, map[string]string{"error": "authentication required"})
		return
	}
	id := householdDevices
	p := s.data.Devices[id]
	now := time.Now()
	p.Screens = slices.DeleteFunc(slices.Clone(p.Screens), func(d pairedScreen) bool { return d.Until <= now.Unix() })
	switch r.Method {
	case "GET":
		devices := make([]map[string]any, 0, len(p.Screens))
		for _, screen := range p.Screens {
			devices = append(devices, map[string]any{"id": secretHash(screen.Hash)[:16], "profile": screen.Profile, "allow_start": screen.AllowStart})
		}
		write(w, 200, map[string]any{"devices": devices})
		return
	case "PATCH":
		var assignment struct {
			Device, Profile string
			AllowStart      *bool `json:"allow_start"`
		}
		if !readInput(w, r, &assignment) {
			return
		}
		if _, ok := s.profile(assignment.Profile); assignment.Profile != "" && !ok {
			http.NotFound(w, r)
			return
		}
		for i, screen := range p.Screens {
			if secretHash(screen.Hash)[:16] == assignment.Device {
				p.Screens[i].Profile = assignment.Profile
				if assignment.AllowStart != nil {
					p.Screens[i].AllowStart = *assignment.AllowStart
				}
				if s.saveDevices(w, id, p) {
					write(w, 200, map[string]bool{"assigned": true})
				}
				return
			}
		}
		http.NotFound(w, r)
		return
	case "DELETE":
		old := s.data.Devices
		s.data.Devices = nil
		if err := s.persist(); err != nil {
			s.data.Devices = old
			write(w, 500, map[string]string{"error": "storage failure"})
			return
		}
		write(w, 200, map[string]bool{"revoked": true})
		return
	}
	var code string
	var input struct {
		Challenge  string
		AllowStart bool `json:"allow_start"`
	}
	if (r.Method == "PUT" || r.ContentLength != 0) && !readInput(w, r, &input) {
		return
	}
	p.AllowStart = input.AllowStart
	if r.Method == "PUT" {
		b, e := hex.DecodeString(input.Challenge)
		if e != nil || len(b) != 32 {
			write(w, 400, map[string]string{"error": "Invalid TV challenge."})
			return
		}
		hash := hex.EncodeToString(b)
		if s.pendingDeviceChallenge(hash, now) {
			write(w, 409, map[string]string{"error": "This screen already has a pending request."})
			return
		}
		for _, v := range s.data.Viewing {
			if v.JoinHash == hash && v.JoinUntil > now.UnixMilli() {
				write(w, 409, map[string]string{"error": "This screen already has a pending request."})
				return
			}
		}
		p.Challenge = hash
	} else {
		var ok bool
		code, ok = newPairingCode(w)
		if !ok {
			return
		}
		p.Challenge = secretHash(code)
	}
	p.Until = now.Add(10 * time.Minute).Unix()
	if s.saveDevices(w, id, p) {
		write(w, 201, map[string]string{"code": code})
	}
}
func (s *Server) joinDevice(w http.ResponseWriter, r *http.Request, hash, normalized string, now time.Time) bool {
	for id, p := range s.data.Devices {
		if (p.Challenge != hash && p.Challenge != normalized) || p.Until <= now.Unix() {
			continue
		}
		if _, ok := s.profile(id); id != householdDevices && !ok {
			continue
		}
		cookie := token()
		const lifetime = 365 * 24 * 60 * 60
		s.finishExchange(w, r, cookie, lifetime, func() bool {
			old := s.data.Devices
			s.data.Devices = maps.Clone(old)
			p.Challenge, p.Until = "", 0
			s.data.Devices[id] = p
			account := s.data.Devices[householdDevices]
			account.Screens = slices.DeleteFunc(slices.Clone(account.Screens), func(d pairedScreen) bool { return d.Until <= now.Unix() })
			account.Screens = append(account.Screens, pairedScreen{AllowStart: p.AllowStart, Profile: id, Hash: secretHash(cookie), Until: now.Unix() + lifetime})
			s.data.Devices[householdDevices] = account
			if err := s.persist(); err != nil {
				s.data.Devices = old
				write(w, 500, map[string]string{"error": "storage failure"})
				return false
			}
			return true
		})
		return true
	}
	return false
}
func (s *Server) findScreen(hash string) (pairedScreen, bool) {
	for _, p := range s.data.Devices {
		for _, d := range p.Screens {
			if d.Hash == hash && d.Until > time.Now().Unix() {
				return d, true
			}
		}
	}
	return pairedScreen{}, false
}
func (s *Server) deviceStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var screen pairedScreen
	var paired bool
	if c, err := r.Cookie("familytv_viewer"); err == nil {
		screen, paired = s.findScreen(secretHash(c.Value))
	}
	label := ""
	if paired {
		label = secretHash(screen.Hash)[:6]
	}
	write(w, 200, map[string]any{"screen": label, "allow_start": paired && screen.AllowStart && screen.Profile != "" && s.data.Account != nil && s.data.Account.PINHash != "", "paired": paired, "active": paired && active(s.data.Viewing[screen.Profile], time.Now())})
}

func (s *Server) deviceStart(w http.ResponseWriter, r *http.Request) {
	var input struct{ PIN string }
	if !readInput(w, r, &input) {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cookie, err := r.Cookie("familytv_viewer")
	if err != nil {
		write(w, 403, map[string]string{"error": "Pair this screen first."})
		return
	}
	screen, ok := s.findScreen(secretHash(cookie.Value))
	profile, assigned := s.profile(screen.Profile)
	if !ok || !assigned || !screen.AllowStart || s.data.Account == nil || s.data.Account.PINHash == "" {
		write(w, 403, map[string]string{"error": "A parent must enable session starts and configure a parent PIN."})
		return
	}
	if !s.checkStartPIN(w, input.PIN) {
		return
	}
	now := time.Now()
	if active(s.data.Viewing[profile.ID], now) {
		write(w, 200, map[string]bool{"active": true})
		return
	}
	v := Viewing{Profile: profile.ID, Mode: "one_video", Last: true, Rotation: token()[:12]}
	if err := s.applyPolicy(&v, profile.Policy, now); err != nil {
		write(w, 409, map[string]string{"error": err.Error()})
		return
	}
	if s.setViewing(w, profile.ID, &v) {
		write(w, 201, map[string]bool{"active": true})
	}
}

// Called with the household lock held. Count attempts across paired screens.
func (s *Server) checkStartPIN(w http.ResponseWriter, pin string) bool {
	const key = "parent-start-pin"
	now := time.Now()
	attempt := s.data.Attempts[key]
	if attempt.Until <= now.Unix() {
		attempt = loginAttempts{}
	}
	if attempt.Count >= 5 {
		write(w, 429, map[string]string{"error": "Try again in five minutes."})
		return false
	}
	if s.data.Account != nil && s.data.Account.PINHash != "" && pin != "" && validPIN(pin) && subtle.ConstantTimeCompare([]byte(digest(pin, "pin:"+s.data.Account.Salt)), []byte(s.data.Account.PINHash)) == 1 {
		return true
	}
	before := s.data.Attempts
	s.data.Attempts = maps.Clone(before)
	if s.data.Attempts == nil {
		s.data.Attempts = map[string]loginAttempts{}
	}
	attempt.Count++
	attempt.Until = now.Add(5 * time.Minute).Unix()
	s.data.Attempts[key] = attempt
	if err := s.persist(); err != nil {
		s.data.Attempts = before
		write(w, 500, map[string]string{"error": "Storage failure."})
		return false
	}
	write(w, 403, map[string]string{"error": "Parent PIN required."})
	return false
}
