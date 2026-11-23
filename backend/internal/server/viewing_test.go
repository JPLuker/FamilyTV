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
