package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDeviceSurvivesSessionAndServerRestart(t *testing.T) {
	s, err := New(filepath.Join(t.TempDir(), "state.json"), false)
	if err != nil {
		t.Fatal(err)
	}
	s.data.Profiles = []ChildProfile{{ID: "child"}}
	s.data.Sessions["parent"] = time.Now().Add(time.Hour).Unix()
	parent := &http.Cookie{Name: "familytv_admin", Value: "parent"}
	call := func(method, path, body string, c *http.Cookie, want int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if c != nil {
			r.AddCookie(c)
		}
		w := httptest.NewRecorder()
		s.Routes().ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("%s %s: %d %s", method, path, w.Code, w.Body.String())
		}
		return w
	}
	const devices = "/api/profiles/child/devices"
	const session = "/api/profiles/child/session"
	call("POST", devices, "", nil, 401)
	w := call("POST", devices, "", parent, 201)
	var issued struct{ Code string }
	if err = json.Unmarshal(w.Body.Bytes(), &issued); err != nil {
		t.Fatal(err)
	}
	body := `{"code":"` + issued.Code + `"}`
	w = call("POST", "/api/watch/join", body, nil, 200)
	child := w.Result().Cookies()[0]
	if child.MaxAge <= 0 {
		t.Fatal("cookie must persist across browser restarts")
	}
	call("POST", "/api/watch/join", body, nil, 403)
	call("POST", devices, "", child, 401)
	call("GET", "/api/watch", "", child, 403)
	call("GET", "/api/watch/media/clip/file", "", child, 404)
	call("POST", session, `{"mode":"manual"}`, parent, 201)
	call("GET", "/api/watch", "", child, 200)
	call("DELETE", session, "", parent, 200)
	call("GET", "/api/watch", "", child, 403)
	s, err = New(s.path, false)
	if err != nil {
		t.Fatal(err)
	}
	call("POST", session, `{"mode":"manual"}`, parent, 201)
	call("GET", "/api/watch", "", child, 200)
	call("DELETE", devices, "", parent, 200)
	call("GET", "/api/watch", "", child, 403)
}
