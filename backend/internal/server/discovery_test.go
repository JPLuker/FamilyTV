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
	first, index, pages := d.page(rows, 0)
	if index != 0 || pages != 4 || len(first) != 4 || first[0].ID != "13" || first[1].ID != "14" || first[2].ID != "0" {
		t.Fatal(first, index, pages)
	}
	last, index, _ := d.page(rows, 9999)
	if index != 3 || len(last) != 3 {
		t.Fatal(last, index)
	}
	if (Discovery{Choices: 7}).valid() || !(Discovery{Preset: "2-4", Search: "on", Choices: 12}).valid() {
		t.Fatal("capability validation")
	}
}
