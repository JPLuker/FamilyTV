package server

import (
	"net/http"
	"slices"
	"strings"
	"time"
)

type ApprovalEvent struct {
	At         int64    `json:"at"`
	Action     string   `json:"action"`
	Titles     []string `json:"titles"`
	ProfileIDs []string `json:"profile_ids"`
	Target     string   `json:"target"`
	Count      int      `json:"count"`
}

func blockedName(list, name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	for entry := range strings.SplitSeq(list, "\n") {
		if strings.EqualFold(strings.TrimSpace(entry), name) {
			return true
		}
	}
	return false
}

func (s *Server) logApproval(action, target string, titles, profileIDs []string) {
	history := s.data.ApprovalLog
	start := max(0, len(history)-199)
	s.data.ApprovalLog = append(slices.Clone(history[start:]), ApprovalEvent{At: time.Now().UnixMilli(), Action: action, Target: target, Titles: titles, ProfileIDs: slices.Clone(profileIDs), Count: len(titles)})
}

func (s *Server) approvals(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.authorized(r) {
		write(w, 401, map[string]string{"error": "Parent sign-in required."})
		return
	}
	if r.Method == "GET" {
		write(w, 200, map[string]any{"events": s.data.ApprovalLog, "profiles": s.data.Profiles})
		return
	}
	var change struct {
		IDs     []string `json:"ids"`
		Profile string   `json:"profile"`
		Action  string   `json:"action"`
	}
	if !readInput(w, r, &change) {
		return
	}
	profile, ok := s.profile(change.Profile)
	if !ok || len(change.IDs) == 0 || len(change.IDs) > 100 || !slices.Contains([]string{"approve", "revoke"}, change.Action) {
		write(w, 400, map[string]string{"error": "Choose a profile, action and up to 100 videos."})
		return
	}
	pending := make(map[string]bool, len(change.IDs))
	for _, id := range change.IDs {
		pending[id] = true
	}
	next := slices.Clone(s.data.Media)
	var titles []string
	for i, m := range next {
		if !pending[m.ID] {
			continue
		}
		delete(pending, m.ID)
		approved := slices.Contains(m.Review.Profiles, profile.ID)
		if approved == (change.Action == "approve") && m.Reviewed {
			continue
		}
		ids := slices.Clone(m.Review.Profiles)
		if change.Action == "approve" && !approved {
			ids = append(ids, profile.ID)
		} else if change.Action == "revoke" {
			ids = slices.DeleteFunc(ids, func(id string) bool { return id == profile.ID })
		}
		next[i].Review.Profiles = ids
		next[i].ReviewVersion++
		next[i].Reviewed = true
		titles = append(titles, m.Title)
	}
	if len(pending) > 0 {
		write(w, 409, map[string]string{"error": "Library changed. Reload and select videos again."})
		return
	}
	count := len(titles)
	if count > 0 {
		before := s.data
		s.data.Media = next
		s.logApproval(change.Action, "Selected videos", titles, []string{profile.ID})
		if err := s.persist(); err != nil {
			s.data = before
			write(w, 500, map[string]string{"error": "Storage failure."})
			return
		}
	}
	write(w, 200, map[string]int{"changed": count})
}

func (s *Server) coverage(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.authorized(r) {
		write(w, 401, map[string]string{"error": "Parent sign-in required."})
		return
	}
	p, ok := s.profile(r.PathValue("id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	excluded := map[string]int{}
	counts := map[string]map[string]int{}
	approved, playable, shortCaptioned := 0, 0, 0
	for _, m := range s.data.Media {
		if !slices.Contains(m.Review.Profiles, p.ID) {
			continue
		}
		approved++
		if reason := exclusionReason(m, p); reason != "" {
			excluded[reason]++
			continue
		}
		if !available(m, p) {
			excluded["Source or required caption file unavailable"]++
			continue
		}
		playable++
		band := "Over 20 minutes"
		if m.Duration < 600 {
			band = "Under 10 minutes"
		} else if m.Duration <= 1200 {
			band = "10–20 minutes"
		}
		caption := m.Review.Captions
		if caption == "verified" {
			f, _, err := openCaption(m.Path)
			if err != nil {
				caption = "missing"
			} else {
				f.Close()
				if m.Duration < 600 {
					shortCaptioned++
				}
			}
		}
		for dimension, value := range map[string]string{"Subject": m.Review.Subject, "Length": band, "Creator": m.Metadata.Creator, "Captions": caption, "Language": m.Metadata.Language, "Learning goal evidence": m.Review.Confidence, "Metadata source": m.Metadata.Source, "Category": m.Review.Category, "Suggested age": m.Review.AgeBand, "Audio description": m.Review.Description} {
			if strings.TrimSpace(value) == "" {
				value = "unknown"
			}
			if counts[dimension] == nil {
				counts[dimension] = map[string]int{}
			}
			counts[dimension][value]++
		}
	}
	write(w, 200, map[string]any{"approved": approved, "playable": playable, "excluded": excluded, "counts": counts, "short_captioned": shortCaptioned})
}
