package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func TestProfileDeletionIsAtomicAndErasesReferences(t *testing.T) {
	s, err := New(filepath.Join(t.TempDir(), "state.json"), false)
	if err != nil {
		t.Fatal(err)
	}
	s.data.Profiles = []ChildProfile{{ID: "p"}}
	s.data.Viewing = map[string]Viewing{"p": {Profile: "p"}}
	s.data.Media = []Media{{Review: Review{Profiles: []string{"p", "keep"}}}}
	s.data.Sessions["parent"] = time.Now().Add(time.Minute).Unix()
	remove := func() int {
		r := httptest.NewRequest("DELETE", "/api/profiles/p", nil)
		r.AddCookie(&http.Cookie{Name: "familytv_admin", Value: "parent"})
		w := httptest.NewRecorder()
		s.Routes().ServeHTTP(w, r)
		return w.Code
	}
	s.save = func(state) error { return errors.New("storage unavailable") }
	if remove() != 500 || len(s.data.Profiles) != 1 || len(s.data.Viewing) != 1 || len(s.data.Media[0].Review.Profiles) != 2 {
		t.Fatal("partial deletion")
	}
	s.save = nil
	if remove() != 200 {
		t.Fatal("remove")
	}
	reloaded, err := New(s.path, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.data.Profiles) != 0 || len(reloaded.data.Viewing) != 0 || len(reloaded.data.Media[0].Review.Profiles) != 1 || reloaded.data.Media[0].Review.Profiles[0] != "keep" {
		t.Fatal("stale profile reference")
	}
}
