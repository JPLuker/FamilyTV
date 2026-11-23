package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestChannelTimingAndGates(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FAMILYTV_MEDIA_DIR", root)
	path := filepath.Join(root, "file.mp4")
	if err := os.WriteFile(path, []byte("video"), 0600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	s, err := New(filepath.Join(t.TempDir(), "state.json"), false)
	if err != nil {
		t.Fatal(err)
	}
	p := ChildProfile{ID: "p"}
	s.data.Profiles = []ChildProfile{p}
	m := Media{ID: "visible", Title: "Approved", Path: "file.mp4", Size: info.Size(), Modified: info.ModTime().UnixNano(), Duration: 60, Review: Review{Profiles: []string{"p"}}}
	hidden := m
	hidden.ID = "hidden"
	hidden.Review = Review{}
	s.data.Media = []Media{m, hidden}
	now := time.Now().UnixMilli()
	c := Channel{ID: "channel", Name: "Nature", Anchor: now, Slots: []Slot{{"visible", 60000}, {"hidden", 30000}}}
	s.data.Channels = []Channel{c}
	v := Viewing{Profile: p.ID, CookieHash: secretHash("viewer")}
	s.data.Viewing = map[string]Viewing{p.ID: v}
	if g := s.guide(c, p, v, now+90001); g.Now == nil || g.Now.Start != now+90000 || g.Next != nil {
		t.Fatal(g)
	}
	if g := s.guide(c, p, v, now+60000); g.Now != nil || g.Next == nil {
		t.Fatal(g)
	}
	v.Deadline = now + 50000
	if g := s.guide(c, p, v, now); g.Now != nil {
		t.Fatal("deadline ignored")
	}
	call := func(method, path, body string, c *http.Cookie) int {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if c != nil {
			r.AddCookie(c)
		}
		s.Routes().ServeHTTP(w, r)
		return w.Code
	}
	viewer := &http.Cookie{Name: "familytv_viewer", Value: "viewer"}
	admin := &http.Cookie{Name: "familytv_admin", Value: "parent"}
	s.data.Sessions["parent"] = time.Now().Add(time.Hour).Unix()
	if call("POST", "/api/channels", `{"name":"New","media":["visible"]}`, viewer) != 401 || call("POST", "/api/channels", `{"name":"New","media":["visible"]}`, admin) != 201 {
		t.Fatal("channel permissions")
	}
	if call("POST", "/api/watch/tune", `{"id":"channel"}`, viewer) != 200 || call("GET", "/api/watch/media/visible/file", "", viewer) != 200 || call("GET", "/api/watch/media/hidden/file", "", viewer) != 404 {
		t.Fatal("stream gates")
	}
	if call("PATCH", "/api/profiles/p/session", "", admin) != 200 || s.data.Viewing["p"].Deadline != now+60000 {
		t.Fatal("stop at program boundary")
	}
	if call("POST", "/api/watch/finish", `{"id":"visible"}`, viewer) != 200 || call("GET", "/api/watch", "", viewer) != 403 {
		t.Fatal("ending")
	}
}
