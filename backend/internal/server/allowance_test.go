package server

import (
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func TestWindowAcrossDST(t *testing.T) {
	s := &Server{}
	cases := []struct {
		start, end, now, stop string
		allow                 bool
	}{
		{"01:00", "03:15", "2026-03-08T01:55:00-05:00", "2026-03-08T03:15:00-04:00", true},
		{"01:20", "01:50", "2026-11-01T01:25:00-05:00", "2026-11-01T01:50:00-05:00", true},
		{"22:00", "07:00", "2026-11-01T23:45:00-05:00", "2026-11-02T07:00:00-05:00", true},
		{"08:00", "09:30", "2026-11-01T09:30:00-05:00", "", false},
	}
	for _, c := range cases {
		v := Viewing{Profile: "test"}
		now, _ := time.Parse(time.RFC3339, c.now)
		err := s.applyPolicy(&v, ViewingPolicy{Start: c.start, End: c.end, Zone: "America/New_York"}, now)
		if (err == nil) != c.allow {
			t.Fatal(c, err)
		}
		if c.allow {
			stop, _ := time.Parse(time.RFC3339, c.stop)
			if v.Deadline != stop.UnixMilli() {
				t.Fatal(c, time.UnixMilli(v.Deadline))
			}
		}
	}
}
func TestDailyAndCategoryBudgetsPersistAcrossHandoff(t *testing.T) {
	s, err := New(filepath.Join(t.TempDir(), "state"), false)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	policy := ViewingPolicy{Daily: 10, Zone: "UTC", Categories: map[string]int{"Animals": 2}}
	v := Viewing{Profile: "p", Mode: "manual"}
	if err = s.applyPolicy(&v, policy, now); err != nil {
		t.Fatal(err)
	}
	// Model a session started 70 seconds ago with this category selected.
	v.Budget.Since = now.Add(-70 * time.Second).UnixMilli()
	v.Budget.Category = "animals"
	v.Budget.CategoryEnd = now.Add(50 * time.Second).UnixMilli()
	v.Deadline = now.Add(530 * time.Second).UnixMilli()
	s.data.Viewing = map[string]Viewing{"p": v}
	long := Media{Duration: 51, Review: Review{Category: "Animals"}}
	if fits(long, v, now) {
		t.Fatal("category limit ignored")
	}
	paired := v
	paired.CookieHash = secretHash("new screen")
	if !s.setViewing(httptest.NewRecorder(), "p", &paired) || paired.cutoff() != v.cutoff() {
		t.Fatal("pairing reset time")
	}
	if !s.setViewing(httptest.NewRecorder(), "p", nil) {
		t.Fatal("stop")
	}
	s, err = New(s.path, false)
	if err != nil {
		t.Fatal(err)
	}
	next := Viewing{Profile: "p"}
	if err = s.applyPolicy(&next, policy, time.Now()); err != nil {
		t.Fatal(err)
	}
	if next.Budget.Remaining["animals"] > 50000 || fits(long, next, time.Now()) || next.Deadline > v.Deadline+2000 {
		t.Fatal("stop/restart replenished allowance", next)
	}
}
