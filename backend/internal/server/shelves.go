package server

import (
	"maps"
	"math"
	"net/http"
	"slices"
	"time"
)

type Bookmark struct {
	Favorite bool    `json:"favorite"`
	Later    bool    `json:"later"`
	Position float64 `json:"position"`
}
type ShelfEntry struct {
	Item     WatchItem `json:"item"`
	Bookmark Bookmark  `json:"bookmark"`
	Playable bool      `json:"playable"`
}

func (s *Server) changeBookmark(profile, id string, next Bookmark) {
	all := maps.Clone(s.data.Bookmarks)
	if all == nil {
		all = map[string]map[string]Bookmark{}
	}
	items := maps.Clone(all[profile])
	if items == nil {
		items = map[string]Bookmark{}
	}
	if next == (Bookmark{}) {
		delete(items, id)
	} else {
		items[id] = next
	}
	all[profile] = items
	s.data.Bookmarks = all
}
func (s *Server) bookmark(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, p, ok := s.viewer(r)
	if !ok {
		write(w, 403, map[string]string{"error": "Session ended."})
		return
	}
	var input struct {
		ID       string  `json:"id"`
		Kind     string  `json:"kind"`
		Enabled  bool    `json:"enabled"`
		Position float64 `json:"position"`
	}
	if !readInput(w, r, &input) {
		return
	}
	if !slices.Contains([]string{"favorite", "later", "position"}, input.Kind) {
		write(w, 400, map[string]string{"error": "Unknown list."})
		return
	}
	for _, m := range s.data.Media {
		if m.ID != input.ID || !available(m, p) {
			continue
		}
		b := s.data.Bookmarks[p.ID][m.ID]
		switch input.Kind {
		case "position":
			if v.Current != m.ID || input.Position < 0 || math.IsNaN(input.Position) || input.Position > m.Duration {
				write(w, 400, map[string]string{"error": "Invalid position."})
				return
			}
			b.Position = input.Position
		case "favorite":
			if !fits(m, v, time.Now()) && input.Enabled {
				http.NotFound(w, r)
				return
			}
			b.Favorite = input.Enabled
		case "later":
			if input.Enabled && !fits(m, v, time.Now()) && v.Current != m.ID {
				http.NotFound(w, r)
				return
			}
			b.Later = input.Enabled
		}
		before := s.data.Bookmarks
		s.changeBookmark(p.ID, m.ID, b)
		if err := s.persist(); err != nil {
			s.data.Bookmarks = before
			write(w, 500, map[string]string{"error": "storage failure"})
			return
		}
		write(w, 200, b)
		return
	}
	http.NotFound(w, r)
}
