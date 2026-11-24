package server

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRetiredChannels(t *testing.T) {
	path := filepath.Join(t.TempDir(), "household.json")
	const old = `{"channels":[{"id":"station"}],"viewing":{"first":{"channel":"station"},"second":{"current":"lesson"}},"media":[{"id":"lesson"}],"bookmarks":{"second":{"lesson":{"favorite":true}}}}`
	if err := os.WriteFile(path, []byte(old), 0600); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		s, err := New(path, false)
		if err != nil {
			t.Fatal(err)
		}
		if len(s.data.Viewing) != 1 || s.data.Viewing["second"].Current != "lesson" || len(s.data.Media) != 1 || !s.data.Bookmarks["second"]["lesson"].Favorite || !strings.Contains(string(s.data.ArchivedChannels), "station") {
			t.Fatal("upgrade lost on-demand state or lineup archive")
		}
		for _, route := range []struct{ method, path string }{{"GET", "/api/channels"}, {"POST", "/api/channels"}, {"DELETE", "/api/channels/station"}, {"POST", "/api/watch/tune"}} {
			w := httptest.NewRecorder()
			s.Routes().ServeHTTP(w, httptest.NewRequest(route.method, route.path, nil))
			if w.Code != 404 {
				t.Errorf("%s %s: %d", route.method, route.path, w.Code)
			}
		}
		if err = s.persist(); err != nil {
			t.Fatal(err)
		}
	}
}
