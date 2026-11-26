package server

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
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
		if id == householdDevices { continue }
		account := st.Devices[householdDevices]
		for _, screen := range old.Screens {
			screen.Profile = id
			account.Screens = append(account.Screens, screen)
		}
		st.Devices[householdDevices] = account
		old.Screens = nil
		if old.Challenge == "" { delete(st.Devices, id) } else { st.Devices[id] = old }
	}
	return nil
}

type DevicePairing struct {
	Challenge string         `json:"challenge,omitempty"`
	Until     int64          `json:"until,omitempty"`
	Screens   []pairedScreen `json:"screens,omitempty"`
}
type pairedScreen struct {
	Profile string `json:"profile,omitempty"`
	Hash  string `json:"hash"`
	Until int64  `json:"until"`
}

func newPairingCode(w http.ResponseWriter) (string, bool) {
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		write(w, 500, map[string]string{"error": "Pairing unavailable."})
		return "", false
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw[:]), true
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
		devices := make([]map[string]string, 0, len(p.Screens))
		for _, screen := range p.Screens {
			devices = append(devices, map[string]string{"id": secretHash(screen.Hash)[:16], "profile": screen.Profile})
		}
		write(w, 200, map[string]any{"devices": devices})
		return
	case "PATCH":
		var assignment struct{ Device, Profile string }
		if !readInput(w, r, &assignment) { return }
		if _, ok := s.profile(assignment.Profile); assignment.Profile != "" && !ok {
			http.NotFound(w, r); return
		}
		for i, screen := range p.Screens {
			if secretHash(screen.Hash)[:16] == assignment.Device {
				p.Screens[i].Profile = assignment.Profile
				if s.saveDevices(w,id,p) { write(w,200,map[string]bool{"assigned":true}) }
				return
			}
		}
		http.NotFound(w,r)
		return
	case "DELETE":
		if s.saveDevices(w, id, DevicePairing{}) {
			write(w, 200, map[string]bool{"revoked": true})
		}
		return
	}
	var code string
	if r.Method == "PUT" {
		var input struct{ Challenge string }
		if !readInput(w, r, &input) {
			return
		}
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
		p.Screens = slices.DeleteFunc(slices.Clone(p.Screens), func(d pairedScreen) bool { return d.Until <= now.Unix() })
		p.Screens = append(p.Screens, pairedScreen{Profile: id, Hash: secretHash(cookie), Until: now.Unix() + lifetime})
		p.Challenge = ""
		p.Until = 0
		s.finishExchange(w, r, cookie, lifetime, func() bool { return s.saveDevices(w, id, p) })
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
	if paired { label = secretHash(screen.Hash)[:6] }
	write(w, 200, map[string]any{"screen": label, "paired": paired, "active": paired && active(s.data.Viewing[screen.Profile], time.Now())})
}
