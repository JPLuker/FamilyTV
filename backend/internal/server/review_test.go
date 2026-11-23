package server

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReviewPermissionsAndEligibility(t *testing.T) {
	s, err := New(filepath.Join(t.TempDir(), "state"), false)
	if err != nil {
		t.Fatal(err)
	}
	s.data.Profiles = []ChildProfile{{ID: "sam", Captions: true, AudioDescription: true}}
	s.data.Media = []Media{{ID: "one", Captions: true}}
	s.data.Sessions["parent"] = time.Now().Add(time.Hour).Unix()
	review := func(auth bool, body string) int {
		r := httptest.NewRequest("PUT", "/api/media/one/review", strings.NewReader(body))
		if auth {
			r.AddCookie(&http.Cookie{Name: "familytv_admin", Value: "parent"})
		}
		w := httptest.NewRecorder()
		s.Routes().ServeHTTP(w, r)
		return w.Code
	}
	complete := `{"profiles":["sam"],"subject":"Forests","caption_quality":"verified","audio_description":"verified"}`
	if eligible(s.data.Media[0], s.data.Profiles[0]) {
		t.Fatal("unreviewed item eligible")
	}
	if review(false, complete) != 401 || review(true, strings.ReplaceAll(complete, "sam", "unknown")) != 400 || review(true, complete) != 200 {
		t.Fatal("review permissions or validation")
	}
	reopened, err := New(s.path, false)
	if err != nil {
		t.Fatal(err)
	}
	if !eligible(reopened.data.Media[0], s.data.Profiles[0]) {
		t.Fatal("review did not persist")
	}
	if review(true, strings.ReplaceAll(complete, "verified", "unknown")) != 200 || eligible(s.data.Media[0], s.data.Profiles[0]) {
		t.Fatal("unverified accessibility")
	}
	if review(true, strings.ReplaceAll(complete, `"sam"`, "")) != 200 || eligible(s.data.Media[0], s.data.Profiles[0]) {
		t.Fatal("revocation")
	}
}
