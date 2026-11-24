package server

import (
	"encoding/json"
	"io"
	"net/http"
	"slices"
	"strings"
)

type Review struct {
	Profiles    []string `json:"profiles"`
	Subject     string   `json:"subject"`
	Description string   `json:"audio_description"`
	Captions    string   `json:"caption_quality"`
}

func (s *Server) reviewMedia(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.authorized(r) {
		write(w, 401, map[string]string{"error": "authentication required"})
		return
	}
	var review Review
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192))
	dec.DisallowUnknownFields()
	if dec.Decode(&review) != nil || dec.Decode(&struct{}{}) != io.EOF {
		write(w, 400, map[string]string{"error": "Invalid review."})
		return
	}
	review.Subject = strings.TrimSpace(review.Subject)
	states := []string{"unknown", "verified", "missing"}
	if len([]rune(review.Subject)) > 100 || !slices.Contains(states, review.Description) || !slices.Contains(states, review.Captions) || len(review.Profiles) > 20 {
		write(w, 400, map[string]string{"error": "Invalid review settings."})
		return
	}
	slices.Sort(review.Profiles)
	review.Profiles = slices.Compact(review.Profiles)
	for _, id := range review.Profiles {
		if !slices.ContainsFunc(s.data.Profiles, func(p ChildProfile) bool { return p.ID == id }) {
			write(w, 400, map[string]string{"error": "A selected profile no longer exists."})
			return
		}
	}
	for i, m := range s.data.Media {
		if m.ID != r.PathValue("id") {
			continue
		}
		if review.Captions == "verified" && !m.Captions {
			write(w, 400, map[string]string{"error": "Import a WebVTT caption file before verifying captions."})
			return
		}
		s.data.Media[i].Review = review
		if err := s.persist(); err != nil {
			s.data.Media[i] = m
			write(w, 500, map[string]string{"error": "storage failure"})
			return
		}
		write(w, 200, s.data.Media[i])
		return
	}
	http.NotFound(w, r)
}
func eligible(m Media, p ChildProfile) bool {
	if p.MaxVideoMinutes > 0 && (m.Duration <= 0 || m.Duration > float64(p.MaxVideoMinutes*60)) {
		return false
	}
	return slices.Contains(m.Review.Profiles, p.ID) && (!p.Captions || (m.Captions && m.Review.Captions == "verified")) && (!p.AudioDescription || m.Review.Description == "verified")
}
