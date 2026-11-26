package server

import (
	"encoding/json"
	"io"
	"net/http"
	"slices"
	"strings"
)

type Review struct {
	AgeBand     string   `json:"age_band,omitempty"`
	Category    string   `json:"category,omitempty"`
	Goal        string   `json:"goal,omitempty"`
	Reviewer    string   `json:"reviewer,omitempty"`
	Reference   string   `json:"reference,omitempty"`
	Confidence  string   `json:"confidence,omitempty"`
	Together    string   `json:"together,omitempty"`
	Activity    string   `json:"activity,omitempty"`
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
	var input struct {
		Review
		ExpectedVersion *uint64 `json:"expected_version,omitempty"`
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384))
	dec.DisallowUnknownFields()
	if dec.Decode(&input) != nil || dec.Decode(&struct{}{}) != io.EOF {
		write(w, 400, map[string]string{"error": "Invalid review."})
		return
	}
	review := input.Review
	review.Subject = strings.TrimSpace(review.Subject)
	states := []string{"unknown", "verified", "missing"}
	for _, field := range []string{review.Category, review.Goal, review.Reviewer, review.Reference, review.Together, review.Activity} {
		if len([]rune(field)) > 400 {
			write(w, 400, map[string]string{"error": "Review fields must be 400 characters or fewer."})
			return
		}
	}
	if !slices.Contains([]string{"", "2-4", "5-7", "8-11", "12-14", "15-17"}, review.AgeBand) || !slices.Contains([]string{"", "described", "source-reviewed"}, review.Confidence) || (review.Confidence == "source-reviewed" && (strings.TrimSpace(review.Reference) == "" || strings.TrimSpace(review.Reviewer) == "")) {
		write(w, 400, map[string]string{"error": "Choose a valid age band and include a reviewer and reference for a source-reviewed goal."})
		return
	}
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
		if input.ExpectedVersion != nil && *input.ExpectedVersion != m.ReviewVersion {
			write(w, 409, map[string]string{"error": "Another review changed this video. Reload the library and reopen its review."})
			return
		}
		if review.Captions == "verified" && !m.Captions {
			write(w, 400, map[string]string{"error": "Import a WebVTT caption file before verifying captions."})
			return
		}
		oldLog := s.data.ApprovalLog
		s.logApproval("Review saved", "Approved for", []string{m.Title}, review.Profiles)
		s.data.Media[i].ReviewVersion++
		s.data.Media[i].CategoryChecked = true
		s.data.Media[i].Reviewed = true
		s.data.Media[i].Review = review
		if err := s.persist(); err != nil {
			s.data.ApprovalLog = oldLog
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
	return exclusionReason(m, p) == ""
}
func exclusionReason(m Media, p ChildProfile) string {
	if !slices.Contains(m.Review.Profiles, p.ID) {
		return "Approval required"
	}
	if blockedName(p.BlockedSeries, m.Metadata.Series) {
		return "Blocked series"
	}
	if blockedName(p.BlockedCreators, m.Metadata.Creator) {
		return "Blocked creator"
	}
	if blockedName(p.BlockedCategories, m.Review.Category) {
		return "Blocked category"
	}
	if p.MaxVideoMinutes > 0 && (m.Duration <= 0 || m.Duration > float64(p.MaxVideoMinutes*60)) {
		return "Over the video length limit"
	}
	if p.Captions && (!m.Captions || m.Review.Captions != "verified") {
		return "Verified captions required"
	}
	if p.AudioDescription && m.Review.Description != "verified" {
		return "Verified audio description required"
	}
	return ""
}
