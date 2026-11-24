package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSavedVideoIsolationAndTimeLimit(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FAMILYTV_MEDIA_DIR", root)
	path := filepath.Join(root, "file.mp4")
	if err := os.WriteFile(path, []byte("sample"), 0600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	s, err := New(filepath.Join(t.TempDir(), "household"), false)
	if err != nil {
		t.Fatal(err)
	}
	s.data.Profiles = []ChildProfile{{ID: "a"}, {ID: "b"}}
	s.data.Viewing = map[string]Viewing{"a": {Profile: "a", CookieHash: secretHash("a")}, "b": {Profile: "b", CookieHash: secretHash("b")}}
	s.data.Media = []Media{{ID: "m", Title: "Lesson", Path: "file.mp4", Size: info.Size(), Modified: info.ModTime().UnixNano(), Duration: 60, Review: Review{Profiles: []string{"a", "b"}}}}
	call := func(profile, method, path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.AddCookie(&http.Cookie{Name: "familytv_viewer", Value: profile})
		w := httptest.NewRecorder()
		s.Routes().ServeHTTP(w, r)
		return w
	}
	count := func(profile string) int {
		w := call(profile, "GET", "/api/watch?list=later", "")
		var result struct{ Items []WatchItem }
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &result) != nil {
			t.Fatal(w.Body.String())
		}
		return len(result.Items)
	}
	if call("a", "POST", "/api/watch/bookmark", `{"id":"m","kind":"later","enabled":true}`).Code != 200 {
		t.Fatal("save")
	}
	s, err = New(s.path, false)
	if err != nil {
		t.Fatal(err)
	}
	if count("a") != 1 || count("b") != 0 {
		t.Fatal("saved list after restart")
	}
	v := s.data.Viewing["a"]
	v.Deadline = time.Now().Add(time.Second).UnixMilli()
	s.data.Viewing["a"] = v
	if count("a") != 0 || call("a", "POST", "/api/watch/play", `{"id":"m"}`).Code != 404 {
		t.Fatal("saved video bypassed deadline")
	}
	v.Deadline = 0
	s.data.Viewing["a"] = v
	s.data.Media[0].Review.Profiles = nil
	if count("a") != 0 || call("a", "POST", "/api/watch/bookmark", `{"id":"m","kind":"later","enabled":true}`).Code != 404 {
		t.Fatal("revoked video exposed")
	}
}
