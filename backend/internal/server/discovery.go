package server

import (
	"crypto/sha256"
	"encoding/binary"
	"math/rand"
	"slices"
	"sort"
	"strings"
)

type Discovery struct {
	UniqueCreators   bool   `json:"unique_creators,omitempty"`
	CreatorCooldown  bool   `json:"creator_cooldown,omitempty"`
	MixSubjects      bool   `json:"mix_subjects,omitempty"`
	PaceSeries       bool   `json:"pace_series,omitempty"`
	Reading          string `json:"reading,omitempty"`
	Preset           string `json:"preset,omitempty"`
	Choices          int    `json:"choices,omitempty"`
	PreferredMinutes int    `json:"preferred_minutes,omitempty"`
	Search           string `json:"search,omitempty"`
	Interests        string `json:"interests,omitempty"`
	SpokenLabels     bool   `json:"spoken_labels,omitempty"`
	WarnMinutes      int    `json:"warn_minutes,omitempty"`
}

func (d Discovery) valid() bool {
	return slices.Contains([]string{"", "brief", "detail"}, d.Reading) && slices.Contains([]string{"", "2-4", "5-7", "8-11", "12-14", "15-17"}, d.Preset) && slices.Contains([]int{0, 4, 6, 8, 12}, d.Choices) && d.PreferredMinutes >= 0 && d.PreferredMinutes <= 240 && slices.Contains([]string{"", "on", "off"}, d.Search) && len([]rune(d.Interests)) <= 250 && d.WarnMinutes >= 0 && d.WarnMinutes <= 15
}
func (d Discovery) board(items []WatchItem, rotation, lastCreator string) []WatchItem {
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
	limit := d.Choices
	if limit == 0 {
		limit = 12
	}
	if len(items) > limit && rotation != "" {
		seed := sha256.Sum256([]byte(rotation))
		random := rand.New(rand.NewSource(int64(binary.BigEndian.Uint64(seed[:8]))))
		random.Shuffle(len(items), func(i, j int) { items[i], items[j] = items[j], items[i] })
	}
	sort.SliceStable(items, func(i, j int) bool { return preferred(items[i]) > preferred(items[j]) })
	if d.MixSubjects {
		items = spread(items, func(m WatchItem) string { return strings.ToLower(m.Subject) })
	}
	if d.PaceSeries {
		items = spread(items, func(m WatchItem) string { return m.series })
	}
	if d.CreatorCooldown && lastCreator != "" {
		sort.SliceStable(items, func(i, j int) bool { return items[i].creator != lastCreator && items[j].creator == lastCreator })
	}
	// Return deferred choices before filling the board, retaining creator grouping.
	sort.SliceStable(items, func(i, j int) bool { return items[i].returning && !items[j].returning })
	if d.UniqueCreators {
		seen := make(map[string]bool)
		items = slices.DeleteFunc(items, func(item WatchItem) bool {
			duplicate := seen[item.creator]
			seen[item.creator] = true
			return duplicate
		})
	}
	return items[:min(limit, len(items))]
}

// Preserve the existing preference order within each group. Unknown labels share a bucket.
func buckets(items []WatchItem, key func(WatchItem) string) [][]WatchItem {
	indexes := map[string]int{}
	var groups [][]WatchItem
	for _, m := range items {
		label := key(m)
		i, ok := indexes[label]
		if !ok {
			i = len(groups)
			indexes[label] = i
			groups = append(groups, nil)
		}
		groups[i] = append(groups[i], m)
	}
	return groups
}
func spread(items []WatchItem, key func(WatchItem) string) []WatchItem {
	groups := buckets(items, key)
	result := make([]WatchItem, 0, len(items))
	for len(groups) > 0 {
		g := groups[0]
		groups = groups[1:]
		result = append(result, g[0])
		if len(g) > 1 {
			groups = append(groups, g[1:])
		}
	}
	return result
}
