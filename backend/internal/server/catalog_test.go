package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOfflineCatalogAndCorrection(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FAMILYTV_MEDIA_DIR", root)
	create := func(path, body string) {
		if err := os.WriteFile(filepath.Join(root, path), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	create("Sesame Street S01E02.mp4", "local")
	create("Sesame Street S01E02.nfo", "<episodedetails><title>Example episode</title><showtitle>Sesame Street</showtitle><season>1</season><episode>2</episode></episodedetails>")
	create("Lesson.webm", "local")
	create("Lesson.info.json", `{"title":"Example lesson","channel":"Crash Course","playlist_title":"Example playlist","id":"local-example"}`)
	episode := detectMetadata("Sesame Street S01E02.mp4")
	lesson := detectMetadata("Lesson.webm")
	if episode.Episode != 2 || episode.Series != "Sesame Street" || episode.Source != "local NFO" || lesson.Creator != "Crash Course" || lesson.Playlist != "Example playlist" {
		t.Fatal(episode, lesson)
	}
	fallback := detectMetadata("My.Series.S00E02.Special.mp4")
	if fallback.Kind != "special" || fallback.Title != "Special" {
		t.Fatal(fallback)
	}
	s, err := New(filepath.Join(t.TempDir(), "state"), false)
	if err != nil {
		t.Fatal(err)
	}
	s.data.Sessions["parent"] = time.Now().Add(time.Hour).Unix()
	s.data.Media = []Media{{ID: "one", Path: "Lesson.webm", Title: lesson.Title, Metadata: lesson, Original: lesson}, {Metadata: episode}}
	h := s.Routes()
	call := func(method, path string, body any) *httptest.ResponseRecorder {
		b, _ := json.Marshal(body)
		r := httptest.NewRequest(method, path, bytes.NewReader(b))
		r.AddCookie(&http.Cookie{Name: "familytv_admin", Value: "parent"})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	w := call("GET", "/api/media/scan", nil)
	if w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte("Lesson.webm")) || bytes.Contains(w.Body.Bytes(), []byte(".nfo")) {
		t.Fatal(w.Body.String())
	}
	gaps := missingEpisodes(s.data.Media)
	if len(gaps) != 1 || gaps[0].From != 1 || gaps[0].To != 1 {
		t.Fatal(gaps)
	}
	lesson.Title = "Corrected"
	if call("PUT", "/api/media/one/metadata", lesson).Code != 200 || s.data.Media[0].Title != "Corrected" {
		t.Fatal("override")
	}
	if call("DELETE", "/api/media/one/metadata", nil).Code != 200 || s.data.Media[0].Title != "Example lesson" {
		t.Fatal("restore")
	}
}
