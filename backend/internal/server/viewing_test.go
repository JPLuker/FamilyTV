package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestViewingSessionBoundaries(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FAMILYTV_MEDIA_DIR", root)
	path := filepath.Join(root, "one.mp4")
	if err := os.WriteFile(path, []byte("media"), 0600); err != nil {
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
	s.data.Profiles = []ChildProfile{{ID: "p"}}
	m := Media{ID: "short", Title: "Approved", Path: "one.mp4", Duration: 1, Size: info.Size(), Modified: info.ModTime().UnixNano(), Review: Review{Profiles: []string{"p"}}}
	long := m
	long.ID = "long"
	long.Title = "Too long"
	long.Duration = 200
	hidden := m
	hidden.ID = "hidden"
	hidden.Title = "Private"
	hidden.Review = Review{}
	s.data.Media = []Media{m, long, hidden}
	s.data.Sessions["admin"] = time.Now().Add(time.Hour).Unix()
	admin := &http.Cookie{Name: "familytv_admin", Value: "admin"}
	h := s.Routes()
	call := func(method, path string, body any, c *http.Cookie) *httptest.ResponseRecorder {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		r := httptest.NewRequest(method, path, bytes.NewReader(b))
		if c != nil {
			r.AddCookie(c)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	start := "/api/profiles/p/session"
	join := "/api/watch/join"
	if call("POST", start, map[string]string{"mode": "manual"}, nil).Code != 401 {
		t.Fatal("anonymous start")
	}
	deadline := time.Now().Add(time.Minute).Format(time.RFC3339)
	w := call("POST", start, map[string]string{"mode": "finish_by", "finish_by": deadline}, admin)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var result struct{ Code string }
	json.Unmarshal(w.Body.Bytes(), &result)
	code := map[string]string{"code": result.Code}
	w = call("POST", join, code, nil)
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	child := w.Result().Cookies()[0]
	if call("POST", join, code, nil).Code != 403 || call("GET", "/api/profiles", nil, child).Code != 401 {
		t.Fatal("token scope")
	}
	w = call("GET", "/api/watch", nil, child)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Approved") || strings.Contains(w.Body.String(), "Private") || strings.Contains(w.Body.String(), "Too long") || strings.Contains(w.Body.String(), m.Path) {
		t.Fatal(w.Body.String())
	}
	for _, id := range []string{"hidden", "long"} {
		if call("POST", "/api/watch/play", map[string]string{"id": id}, child).Code != 404 {
			t.Fatal(id)
		}
	}
	if call("GET", "/api/watch/media/short/file", nil, child).Code != 404 {
		t.Fatal("media before selection")
	}
	choice := map[string]string{"id": "short"}
	if call("POST", "/api/watch/play", choice, child).Code != 200 || call("GET", "/api/watch/media/short/file", nil, child).Code != 200 {
		t.Fatal("play")
	}
	if call("PATCH", start, nil, admin).Code != 200 || call("POST", "/api/watch/finish", choice, child).Code != 200 || call("GET", "/api/watch", nil, child).Code != 403 {
		t.Fatal("after current")
	}
	for _, mode := range []string{"duration", "manual", "one_video"} {
		w = call("POST", start, map[string]any{"mode": mode, "minutes": 2}, admin)
		if w.Code != 201 {
			t.Fatal(mode, w.Body.String())
		}
		if mode == "duration" {
			v := s.data.Viewing["p"]
			v.Deadline = time.Now().Add(-time.Second).UnixMilli()
			v.CookieHash = secretHash(child.Value)
			v.Current = "short"
			s.data.Viewing["p"] = v
			if call("GET", "/api/watch/media/short/file", nil, child).Code != 404 || call("GET", "/api/watch", nil, child).Code != 403 {
				t.Fatal("past deadline")
			}
		}
		if call("DELETE", start, nil, admin).Code != 200 {
			t.Fatal(mode)
		}
	}
}

func TestActiveResponseChecksRevocation(t *testing.T) {
	eligible := true
	guarded := checkedStream{ReadSeeker: strings.NewReader("sample"), check: func() bool { return eligible }}
	var b [1]byte
	if n, err := guarded.Read(b[:]); err != nil || n != 1 {
		t.Fatal(err, n)
	}
	eligible = false
	guarded.next = time.Now().Add(-time.Second)
	if n, err := guarded.Read(b[:]); err == nil || n != 0 {
		t.Fatal("stream continued after revocation")
	}
}

func TestIndependentProfilePolicyAndRevocation(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FAMILYTV_MEDIA_DIR", root)
	path := filepath.Join(root, "lesson.mp4")
	if err := os.WriteFile(path, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	s, err := New(filepath.Join(t.TempDir(), "state"), false)
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Hour).UnixMilli()
	s.data.Profiles = []ChildProfile{{ID: "short", MaxVideoMinutes: 1}, {ID: "long", Discovery: Discovery{PreferredMinutes: 1}}}
	s.data.Media = []Media{{ID: "lesson", Title: "Lesson", Duration: 61, Path: "lesson.mp4", Size: info.Size(), Modified: info.ModTime().UnixNano(), Review: Review{Profiles: []string{"short", "long"}}}}
	s.data.Viewing = map[string]Viewing{}
	for _, p := range s.data.Profiles {
		s.data.Viewing[p.ID] = Viewing{Profile: p.ID, CookieHash: secretHash(p.ID), Deadline: deadline, JoinHash: secretHash("pending-" + p.ID), JoinUntil: deadline}
	}
	s.data.Sessions["parent"] = time.Now().Add(time.Hour).Unix()
	admin := &http.Cookie{Name: "familytv_admin", Value: "parent"}
	child := func(id string) *http.Cookie { return &http.Cookie{Name: "familytv_viewer", Value: id} }
	routes := s.Routes()
	call := func(cookie *http.Cookie, method, path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.AddCookie(cookie)
		w := httptest.NewRecorder()
		routes.ServeHTTP(w, r)
		return w
	}
	for _, p := range s.data.Profiles {
		viewer := child(p.ID)
		catalog := call(viewer, "GET", "/api/watch", "")
		if catalog.Code != 200 || strings.Contains(catalog.Body.String(), "Lesson") != (p.ID == "long") {
			t.Fatal(p.ID, catalog.Body.String())
		}
		played := call(viewer, "POST", "/api/watch/play", `{"id":"lesson"}`).Code
		if (p.ID == "short" && played != 404) || (p.ID == "long" && played != 200) {
			t.Fatal(p.ID, played)
		}
		for _, route := range []struct{ method, path string }{{"GET", "/api/admin"}, {"PUT", "/api/admin/pin"}, {"POST", "/api/profiles"}, {"GET", "/api/profiles/export"}, {"GET", "/api/media"}, {"PUT", "/api/media/lesson/review"}, {"DELETE", "/api/profiles/long/session/pair"}} {
			if w := call(viewer, route.method, route.path, "{}"); w.Code != 401 {
				t.Fatal(route, w.Code)
			}
		}
	}
	if w := call(admin, "DELETE", "/api/profiles/long/session/pair", ""); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if call(child("long"), "GET", "/api/watch", "").Code != 403 || call(child("long"), "POST", "/api/watch/join", `{"code":"pending-long"}`).Code != 403 || s.data.Viewing["long"].Deadline != deadline || call(child("short"), "GET", "/api/watch", "").Code != 200 {
		t.Fatal("revocation scope or timing")
	}
}

func TestOpeningViewerLinkSignsOutSharedParentBrowser(t *testing.T) {
	s, err := New(filepath.Join(t.TempDir(), "household.json"), false)
	if err != nil {
		t.Fatal(err)
	}
	expiry := time.Now().Add(time.Hour)
	s.data.Sessions["shared"] = expiry.Unix()
	s.data.Sessions["phone"] = expiry.Unix()
	s.data.Profiles = []ChildProfile{{ID: "p"}}
	s.data.Viewing = map[string]Viewing{"p": {Profile: "p", JoinHash: secretHash("link"), JoinUntil: expiry.UnixMilli()}}
	r := httptest.NewRequest("POST", "/api/watch/join", strings.NewReader(`{"code":"link"}`))
	r.AddCookie(&http.Cookie{Name: "familytv_admin", Value: "shared"})
	w := httptest.NewRecorder()
	s.Routes().ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	saved, err := New(s.path, false)
	if err != nil {
		t.Fatal(err)
	}
	if saved.data.Sessions["shared"] != 0 || saved.data.Sessions["phone"] != expiry.Unix() || saved.data.Viewing["p"].CookieHash == "" {
		t.Fatal("shared browser retained parent access or another device lost it")
	}
}
