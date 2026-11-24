package server

import (
	"math"
	"net/http"
	"strings"
	"time"
)

type Slot struct {
	Media  string `json:"media"`
	Millis int64  `json:"milliseconds"`
}
type Channel struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Anchor int64  `json:"anchor"`
	Slots  []Slot `json:"slots"`
}
type Airing struct {
	Item  WatchItem `json:"item"`
	Start int64     `json:"start"`
	End   int64     `json:"end"`
}
type Guide struct {
	ID   string  `json:"id"`
	Name string  `json:"name"`
	Now  *Airing `json:"now"`
	Next *Airing `json:"next"`
}

func (s *Server) parentChannels(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.authorized(r) {
		write(w, 401, map[string]string{"error": "authentication required"})
		return
	}
	if r.Method == "GET" {
		write(w, 200, map[string]any{"channels": append([]Channel{}, s.data.Channels...)})
		return
	}
	before := s.data.Channels
	if r.Method == "DELETE" {
		for i, c := range before {
			if c.ID == r.PathValue("id") {
				s.data.Channels = append(append([]Channel(nil), before[:i]...), before[i+1:]...)
				if err := s.persist(); err != nil {
					s.data.Channels = before
					write(w, 500, map[string]string{"error": "storage failure"})
					return
				}
				write(w, 200, map[string]bool{"removed": true})
				return
			}
		}
		http.NotFound(w, r)
		return
	}
	var input struct {
		Name  string   `json:"name"`
		Media []string `json:"media"`
	}
	if !readInput(w, r, &input) {
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || len([]rune(input.Name)) > 80 || len(input.Media) == 0 || len(input.Media) > 200 || len(before) >= 100 {
		write(w, 400, map[string]string{"error": "Use a channel name and 1–200 programs. Up to 100 channels are supported."})
		return
	}
	c := Channel{ID: token(), Name: input.Name, Anchor: time.Now().UnixMilli()}
	for _, id := range input.Media {
		found := false
		for _, m := range s.data.Media {
			if m.ID == id && m.Duration > 0 && m.Duration < 86400*30 {
				c.Slots = append(c.Slots, Slot{id, int64(math.Ceil(m.Duration * 1000))})
				found = true
				break
			}
		}
		if !found {
			write(w, 400, map[string]string{"error": "Program unavailable."})
			return
		}
	}
	s.data.Channels = append(before, c)
	if err := s.persist(); err != nil {
		s.data.Channels = before
		write(w, 500, map[string]string{"error": "storage failure"})
		return
	}
	write(w, 201, c)
}

// Durations remain fixed when approval changes, so every device shares the same schedule.
func (c Channel) slotAt(now int64) (int, int64) {
	var total int64
	for _, slot := range c.Slots {
		total += slot.Millis
	}
	if total <= 0 {
		return -1, 0
	}
	offset := ((now-c.Anchor)%total + total) % total
	for i, slot := range c.Slots {
		if offset < slot.Millis {
			return i, now - offset
		}
		offset -= slot.Millis
	}
	return -1, 0
}
func (s *Server) guide(c Channel, p ChildProfile, v Viewing, now int64) Guide {
	g := Guide{ID: c.ID, Name: c.Name}
	i, start := c.slotAt(now)
	if i < 0 {
		return g
	}
	airing := func(slot Slot, start int64) *Airing {
		end := start + slot.Millis
		if v.Deadline > 0 && end > v.Deadline {
			return nil
		}
		for _, m := range s.data.Media {
			if m.ID == slot.Media && available(m, p) {
				return &Airing{watchItem(m), start, end}
			}
		}
		return nil
	}
	g.Now = airing(c.Slots[i], start)
	g.Next = airing(c.Slots[(i+1)%len(c.Slots)], start+c.Slots[i].Millis)
	return g
}
func (s *Server) currentGuide(p ChildProfile, v Viewing, now int64) *Guide {
	for _, c := range s.data.Channels {
		if c.ID == v.Channel {
			g := s.guide(c, p, v, now)
			return &g
		}
	}
	return nil
}
func (s *Server) tune(w http.ResponseWriter, r *http.Request) {
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
	if v.Current != "" || v.Channel != "" {
		write(w, 409, map[string]string{"error": "Finish the current program first."})
		return
	}
	v.Channel = input.ID
	g := s.currentGuide(p, v, time.Now().UnixMilli())
	if g == nil || g.Now == nil {
		http.NotFound(w, r)
		return
	}
	if v.Last {
		v.Deadline = g.Now.End
	}
	if s.setViewing(w, p.ID, &v) {
		write(w, 200, map[string]bool{"tuned": true})
	}
}
