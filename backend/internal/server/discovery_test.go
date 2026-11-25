package server

import (
	"fmt"
	"testing"
)

func TestDiscoveryPreferencesAndSingleBoard(t *testing.T) {
	var rows []WatchItem
	for i := 0; i < 15; i++ {
		rows = append(rows, WatchItem{ID: fmt.Sprint(i), Subject: "Other", Duration: 600})
	}
	rows[13].Subject = "Nature"
	rows[14].Duration = 30
	d := Discovery{Choices: 4, Interests: "nature", PreferredMinutes: 2}
	first := d.board(rows, "", "")
	if len(first) != 4 || first[0].ID != "13" || first[1].ID != "14" || first[2].ID != "0" {
		t.Fatal(first)
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
	items := settings.board(rows, "rotation", "0")
	creators := map[string]bool{}
	if len(items) != 12 || items[0].creator == "0" {
		t.Fatal(items)
	}
	for _, m := range items {
		if creators[m.creator] {
			t.Fatal("duplicate", m)
		}
		creators[m.creator] = true
	}
	items = settings.board([]WatchItem{{creator: "0", ID: "a"}, {creator: "0", ID: "b"}}, "rotation", "0")
	if len(items) != 1 || items[0].creator != "0" {
		t.Fatal("single eligible creator must remain available", items)
	}
}

func TestConfiguredChoiceCounts(t *testing.T) {
	items := make([]WatchItem, 15)
	for _, limit := range []int{4, 6, 8, 12} {
		page := (Discovery{Choices: limit}).board(items, "", "")
		if len(page) != limit {
			t.Fatal(limit, len(page))
		}
	}
}

func TestReturningSavedVideoFitsBoard(t *testing.T) {
	rows := []WatchItem{
		{ID: "a", creator: "first"}, {ID: "b", creator: "second"},
		{ID: "c", creator: "third"}, {ID: "d", creator: "fourth"},
		{ID: "saved", creator: "first", returning: true},
	}
	d := Discovery{Choices: 4, UniqueCreators: true}
	first := d.board(rows, "", "")
	if len(first) != 4 || first[0].ID != "saved" {
		t.Fatalf("saved choice missed the board: %v", first)
	}
	for _, item := range first[1:] {
		if item.creator == "first" {
			t.Fatal("saved choice bypassed creator variety")
		}
	}
}
