package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestParentMedia(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("requires ffmpeg")
	}
	root := t.TempDir()
	t.Setenv("FAMILYTV_MEDIA_DIR", root)
	path := filepath.Join(root, "sample.mp4")
	if output, err := exec.Command("ffmpeg", "-loglevel", "error", "-f", "lavfi", "-i", "color=c=blue:s=32x32:d=1", "-c:v", "mpeg4", path).CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, output)
	}
	if err := os.WriteFile(filepath.Join(root, "sample.vtt"), []byte("WEBVTT\n\n00:00:00.000 --> 00:00:01.000\nBlue\n"), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := New(filepath.Join(t.TempDir(), "state.json"), false)
	if err != nil {
		t.Fatal(err)
	}
	s.data.Sessions["parent"] = time.Now().Add(time.Hour).Unix()
	h := s.Routes()
	call := func(method, path, body string, auth bool, span string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		if auth {
			r.AddCookie(&http.Cookie{Name: "familytv_admin", Value: "parent"})
		}
		if span != "" {
			r.Header.Set("Range", span)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	body := `{"path":"sample.mp4"}`
	if w := call("POST", "/api/media", body, false, ""); w.Code != 401 {
		t.Fatal(w.Code)
	}
	if w := call("POST", "/api/media", `{"path":"../sample.mp4"}`, true, ""); w.Code != 400 {
		t.Fatal(w.Code)
	}
	w := call("POST", "/api/media", body, true, "")
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var m Media
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if m.Title != "sample" || !m.Captions || m.Duration < 0.9 {
		t.Fatal(m)
	}
	if w := call("POST", "/api/media", body, true, ""); w.Code != 409 {
		t.Fatal(w.Code)
	}
	route := "/api/media/" + m.ID
	if w := call("GET", route+"/file", "", false, ""); w.Code != 401 {
		t.Fatal(w.Code)
	}
	if w := call("GET", route+"/file", "", true, "bytes=0-7"); w.Code != 206 || w.Body.Len() != 8 {
		t.Fatal(w.Code, w.Body.Len())
	}
	if w := call("GET", route+"/captions", "", true, ""); w.Code != 200 || !bytes.HasPrefix(w.Body.Bytes(), []byte("WEBVTT")) {
		t.Fatal(w.Code)
	}
	if err := os.WriteFile(path, []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if w := call("GET", route+"/file", "", true, ""); w.Code != 409 {
		t.Fatal(w.Code)
	}
	if w := call("DELETE", route, "", true, ""); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("source file removed", err)
	}
	if w := call("GET", route+"/file", "", true, ""); w.Code != 404 {
		t.Fatal(w.Code)
	}
}
