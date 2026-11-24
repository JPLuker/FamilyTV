package server

import (
	"encoding/json"
	"io"
	"maps"
	"net/http"
	"slices"
	"strings"
)

type ChildProfile struct {
	Avatar           string    `json:"avatar,omitempty"`
	MaxVideoMinutes  int       `json:"max_video_minutes,omitempty"`
	Discovery        Discovery `json:"discovery"`
	LargeText        bool      `json:"large_text"`
	HighContrast     bool      `json:"high_contrast"`
	EndingMusic      bool      `json:"ending_music"`
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Captions         bool      `json:"captions"`
	ReducedMotion    bool      `json:"reduced_motion"`
	AudioDescription bool      `json:"audio_description"`
	Navigation       string    `json:"navigation"`
}

func (s *Server) listProfiles(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.authorized(r) {
		write(w, 401, map[string]string{"error": "authentication required"})
		return
	}
	profiles := append([]ChildProfile{}, s.data.Profiles...)
	write(w, 200, map[string]any{"profiles": profiles})
}
func (s *Server) createProfile(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.authorized(r) {
		write(w, 401, map[string]string{"error": "authentication required"})
		return
	}
	p, ok := readProfile(w, r)
	if !ok {
		return
	}
	if len(s.data.Profiles) >= 20 {
		write(w, 400, map[string]string{"error": "household profile limit reached"})
		return
	}
	p.ID = token()
	s.data.Profiles = append(s.data.Profiles, p)
	if err := s.persist(); err != nil {
		s.data.Profiles = s.data.Profiles[:len(s.data.Profiles)-1]
		write(w, 500, map[string]string{"error": "storage failure"})
		return
	}
	write(w, 201, p)
}
func (s *Server) deleteProfile(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.authorized(r) {
		write(w, 401, map[string]string{"error": "authentication required"})
		return
	}
	id := r.PathValue("id")
	for i, p := range s.data.Profiles {
		if p.ID == id {
			before := s.data
			s.data.Profiles = slices.Delete(slices.Clone(before.Profiles), i, i+1)
			s.data.Viewing = maps.Clone(before.Viewing)
			delete(s.data.Viewing, id)
			s.data.Media = slices.Clone(before.Media)
			for j, m := range s.data.Media {
				if slices.Contains(m.Review.Profiles, id) {
					s.data.Media[j].Review.Profiles = slices.DeleteFunc(slices.Clone(m.Review.Profiles), func(value string) bool { return value == id })
				}
			}
			if err := s.persist(); err != nil {
				s.data = before
				write(w, 500, map[string]string{"error": "storage failure"})
				return
			}
			write(w, 200, map[string]bool{"deleted": true})
			return
		}
	}
	write(w, 404, map[string]string{"error": "profile not found"})
}

func readProfile(w http.ResponseWriter, r *http.Request) (ChildProfile, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	var p ChildProfile
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil || dec.Decode(&struct{}{}) != io.EOF {
		write(w, 400, map[string]string{"error": "invalid profile"})
		return p, false
	}
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" || len([]rune(p.Name)) > 50 || p.ID != "" {
		write(w, 400, map[string]string{"error": "enter a profile name of 1 to 50 characters"})
		return p, false
	}
	if p.Navigation == "" {
		p.Navigation = "visual"
	}
	if p.Navigation != "visual" && p.Navigation != "visual-text" && p.Navigation != "text" {
		write(w, 400, map[string]string{"error": "invalid navigation mode"})
		return p, false
	}
	if !slices.Contains([]string{"", "sun", "moon", "star", "leaf", "fish"}, p.Avatar) || p.MaxVideoMinutes < 0 || p.MaxVideoMinutes > 1440 {
		write(w, 400, map[string]string{"error": "Invalid avatar or maximum video length."})
		return p, false
	}
	if !p.Discovery.valid() {
		write(w, 400, map[string]string{"error": "Invalid discovery settings."})
		return p, false
	}
	return p, true
}
func (s *Server) updateProfile(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.authorized(r) {
		write(w, 401, map[string]string{"error": "authentication required"})
		return
	}
	p, ok := readProfile(w, r)
	if !ok {
		return
	}
	for i, old := range s.data.Profiles {
		if old.ID == r.PathValue("id") {
			p.ID = old.ID
			s.data.Profiles[i] = p
			if err := s.persist(); err != nil {
				s.data.Profiles[i] = old
				write(w, 500, map[string]string{"error": "storage failure"})
				return
			}
			write(w, 200, p)
			return
		}
	}
	write(w, 404, map[string]string{"error": "profile not found"})
}
func (s *Server) exportProfiles(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.authorized(r) {
		write(w, 401, map[string]string{"error": "authentication required"})
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="familytv-profiles.json"`)
	write(w, 200, map[string]any{"version": 1, "profiles": append([]ChildProfile{}, s.data.Profiles...)})
}
