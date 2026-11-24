package server

import (
	"errors"
	"maps"
	"strings"
	"time"
	_ "time/tzdata"
)

type ViewingPolicy struct {
	Zone       string         `json:"zone,omitempty"`
	Start      string         `json:"start,omitempty"`
	End        string         `json:"end,omitempty"`
	Daily      int            `json:"daily,omitempty"`
	Categories map[string]int `json:"categories,omitempty"`
}
type DailyUsage struct {
	Day        string           `json:"day"`
	Total      int64            `json:"total_ms"`
	Categories map[string]int64 `json:"category_ms,omitempty"`
}
type SessionBudget struct {
	Day         string           `json:"day"`
	Since       int64            `json:"since"`
	Remaining   map[string]int64 `json:"remaining,omitempty"`
	Category    string           `json:"category,omitempty"`
	CategoryEnd int64            `json:"category_end,omitempty"`
}

func (p ViewingPolicy) valid() bool {
	if p.Daily < 0 || p.Daily > 1440 || len(p.Categories) > 30 {
		return false
	}
	if p.Zone != "" {
		if _, e := time.LoadLocation(p.Zone); e != nil || p.Zone == "Local" {
			return false
		}
	}
	if p.Start != "" || p.End != "" {
		for _, v := range []string{p.Start, p.End} {
			t, e := time.Parse("15:04", v)
			if e != nil || t.Format("15:04") != v {
				return false
			}
		}
		if p.Start == p.End {
			return false
		}
	}
	seen := map[string]bool{}
	for name, limit := range p.Categories {
		key := strings.ToLower(strings.TrimSpace(name))
		if key == "" || len([]rune(key)) > 400 || seen[key] || limit < 1 || limit > 1440 {
			return false
		}
		seen[key] = true
	}
	return true
}
func earliest(a, b int64) int64 {
	if a == 0 {
		return b
	}
	if b == 0 {
		return a
	}
	return min(a, b)
}
func (v Viewing) cutoff() int64 { return earliest(v.Deadline, v.Budget.CategoryEnd) }

// Evaluate local clock minutes on the actual timeline so skipped and repeated hours
// obey the household's wall-clock window. A session stops at the first closed minute.
func (p ViewingPolicy) permits(t time.Time) bool {
	h := t.Format("15:04")
	if p.Start < p.End {
		return h >= p.Start && h < p.End
	}
	return h >= p.Start || h < p.End
}
func (s *Server) usageAt(id string, now time.Time) DailyUsage {
	u := s.data.Usage[id]
	v := s.data.Viewing[id]
	b := v.Budget
	if b.Day == "" {
		return u
	}
	if u.Day != b.Day {
		u = DailyUsage{Day: b.Day}
	}
	end := earliest(now.UnixMilli(), v.cutoff())
	elapsed := max(int64(0), end-b.Since)
	u.Total += elapsed
	if b.Category != "" {
		u.Categories = maps.Clone(u.Categories)
		if u.Categories == nil {
			u.Categories = map[string]int64{}
		}
		u.Categories[b.Category] += elapsed
	}
	return u
}
func (s *Server) applyPolicy(v *Viewing, p ViewingPolicy, now time.Time) error {
	if p.Daily == 0 && len(p.Categories) == 0 && p.Start == "" {
		return nil
	}
	loc, err := time.LoadLocation(p.Zone)
	if err != nil {
		return err
	}
	local := now.In(loc)
	day := local.Format("2006-01-02")
	if p.Start != "" {
		if !p.permits(local) {
			return errors.New("outside the profile’s viewing window")
		}
		for t := local.Truncate(time.Minute).Add(time.Minute); t.Before(local.Add(49 * time.Hour)); t = t.Add(time.Minute) {
			if !p.permits(t) {
				v.Deadline = earliest(v.Deadline, t.UnixMilli())
				break
			}
		}
	}
	if p.Daily == 0 && len(p.Categories) == 0 {
		return nil
	}
	// Daily allowances reset at the next local date. Start a new session after midnight.
	for t := local.Truncate(time.Minute).Add(time.Minute); ; t = t.Add(time.Minute) {
		if t.Format("2006-01-02") != day {
			v.Deadline = earliest(v.Deadline, t.UnixMilli())
			break
		}
	}
	u := s.usageAt(v.Profile, now)
	if u.Day != day {
		u = DailyUsage{}
	}
	if p.Daily > 0 {
		remaining := int64(p.Daily)*60000 - u.Total
		if remaining <= 0 {
			return errors.New("daily viewing allowance is used up")
		}
		v.Deadline = earliest(v.Deadline, now.UnixMilli()+remaining)
	}
	v.Budget = SessionBudget{Day: day, Since: now.UnixMilli(), Remaining: map[string]int64{}}
	for name, minutes := range p.Categories {
		key := strings.ToLower(strings.TrimSpace(name))
		v.Budget.Remaining[key] = max(0, int64(minutes)*60000-u.Categories[key])
	}
	return nil
}
func (v Viewing) categoryLeft(m Media, now time.Time) (int64, bool) {
	key := strings.ToLower(strings.TrimSpace(m.Review.Category))
	left, limited := v.Budget.Remaining[key]
	if limited && key == v.Budget.Category {
		left = min(left, v.Budget.CategoryEnd-now.UnixMilli())
	}
	return left, limited
}
