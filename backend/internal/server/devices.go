package server

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/hex"
	"net/http"
	"slices"
	"time"
)

type DevicePairing struct {
	Challenge string         `json:"challenge,omitempty"`
	Until     int64          `json:"until,omitempty"`
	Screens   []pairedScreen `json:"screens,omitempty"`
}
type pairedScreen struct {
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
func (s *Server) profileDevices(w http.ResponseWriter, r *http.Request) {
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
	p := s.data.Devices[id]
	now := time.Now()
	p.Screens = slices.DeleteFunc(slices.Clone(p.Screens), func(d pairedScreen) bool { return d.Until <= now.Unix() })
	switch r.Method {
	case "GET":
		write(w, 200, map[string]int{"count": len(p.Screens)})
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
		if _, ok := s.profile(id); !ok {
			continue
		}
		cookie := token()
		const lifetime = 365 * 24 * 60 * 60
		p.Screens = slices.DeleteFunc(slices.Clone(p.Screens), func(d pairedScreen) bool { return d.Until <= now.Unix() })
		p.Screens = append(p.Screens, pairedScreen{Hash: secretHash(cookie), Until: now.Unix() + lifetime})
		p.Challenge = ""
		p.Until = 0
		s.finishExchange(w, r, cookie, lifetime, func() bool { return s.saveDevices(w, id, p) })
		return true
	}
	return false
}
func (s *Server) deviceProfile(hash string) string {
	for id, p := range s.data.Devices {
		for _, d := range p.Screens {
			if d.Hash == hash && d.Until > time.Now().Unix() {
				return id
			}
		}
	}
	return ""
}
func (s *Server) deviceStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := ""
	if c, e := r.Cookie("familytv_viewer"); e == nil {
		id = s.deviceProfile(secretHash(c.Value))
	}
	write(w, 200, map[string]bool{"paired": id != "", "active": id != "" && active(s.data.Viewing[id], time.Now())})
}
