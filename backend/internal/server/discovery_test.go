package server

import (
	"fmt"
	"testing"
)

func TestDiscoveryPreferencesAndStablePages(t *testing.T) {
	var rows []WatchItem
	for i := 0; i < 15; i++ {
		rows = append(rows, WatchItem{ID: fmt.Sprint(i), Subject: "Other", Duration: 600})
	}
	rows[13].Subject = "Nature"
	rows[14].Duration = 30
	d := Discovery{Choices: 4, Interests: "nature", PreferredMinutes: 2}
	first, index, pages := d.page(rows, 0, "")
	if index != 0 || pages != 4 || len(first) != 4 || first[0].ID != "13" || first[1].ID != "14" || first[2].ID != "0" {
		t.Fatal(first, index, pages)
	}
	last, index, _ := d.page(rows, 9999, "")
	if index != 3 || len(last) != 3 {
		t.Fatal(last, index)
	}
	if (Discovery{Choices: 7}).valid() || !(Discovery{Preset: "2-4", Search: "on", Choices: 12}).valid() {
		t.Fatal("capability validation")
	}
}

func TestTwelveCreatorChoicesAndCooldownFallback(t *testing.T) {
	var rows []WatchItem
	for i := 0; i < 12; i++ {
		for j := 0; j < 2; j++ {
			rows = append(rows, WatchItem{ID: fmt.Sprintf("%d/%d", i, j), creator: fmt.Sprint(i)})
		}
	}
	settings := Discovery{Choices: 12, UniqueCreators: true, CreatorCooldown: true}
	seen := map[string]bool{}
	for page := 0; page < 2; page++ {
		items, index, pages := settings.page(rows, page, "0")
		creators := map[string]bool{}
		if pages != 2 || index != page || len(items) != 12 || items[0].creator == "0" {
			t.Fatal(index, pages, items)
		}
		for _, m := range items {
			if creators[m.creator] || seen[m.ID] {
				t.Fatal("duplicate", m)
			}
			creators[m.creator] = true
			seen[m.ID] = true
		}
	}
	items, _, pages := settings.page([]WatchItem{{creator: "0", ID: "a"}, {creator: "0", ID: "b"}}, 0, "0")
	if pages != 2 || len(items) != 1 || items[0].creator != "0" {
		t.Fatal("single eligible creator must remain available", items)
	}
}
