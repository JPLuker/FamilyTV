package server

import (
	"slices"
	"sort"
	"strings"
)

type Discovery struct {
	Preset           string `json:"preset,omitempty"`
	Choices          int    `json:"choices,omitempty"`
	PreferredMinutes int    `json:"preferred_minutes,omitempty"`
	Search           string `json:"search,omitempty"`
	Interests        string `json:"interests,omitempty"`
	SpokenLabels     bool   `json:"spoken_labels,omitempty"`
	WarnMinutes      int    `json:"warn_minutes,omitempty"`
}

func (d Discovery) valid() bool {
	return slices.Contains([]string{"", "2-4", "5-7", "8-11", "12-14", "15-17"}, d.Preset) && slices.Contains([]int{0, 4, 6, 8, 12}, d.Choices) && d.PreferredMinutes >= 0 && d.PreferredMinutes <= 240 && slices.Contains([]string{"", "on", "off"}, d.Search) && len([]rune(d.Interests)) <= 250 && d.WarnMinutes >= 0 && d.WarnMinutes <= 15
}
func (d Discovery) page(items []WatchItem, page int) ([]WatchItem, int, int) {
	topics := strings.Split(strings.ToLower(d.Interests), ",")
	preferred := func(m WatchItem) int {
		score := 0
		for _, topic := range topics {
			if t := strings.TrimSpace(topic); t != "" && strings.EqualFold(t, m.Subject) {
				score += 2
				break
			}
		}
		if d.PreferredMinutes > 0 && m.Duration <= float64(d.PreferredMinutes*60) {
			score++
		}
		return score
	}
	sort.SliceStable(items, func(i, j int) bool { return preferred(items[i]) > preferred(items[j]) })
	limit := d.Choices
	if limit == 0 {
		limit = 12
	}
	pages := max(1, (len(items)+limit-1)/limit)
	page = max(0, min(page, pages-1))
	start := page * limit
	return items[start:min(start+limit, len(items))], page, pages
}
