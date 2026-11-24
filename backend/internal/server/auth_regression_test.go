package server

import (
	"crypto/pbkdf2"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestOlderPasswordDigest(t *testing.T) {
	key, err := pbkdf2.Key(sha256.New, "previous password", []byte("example encoded salt"), 310000, 32)
	if err != nil || digest("previous password", "example encoded salt") != base64.RawStdEncoding.EncodeToString(key) {
		t.Fatal("existing credential format changed")
	}
}

func TestMutationsValidateOrigin(t *testing.T) {
	s, err := New(filepath.Join(t.TempDir(), "state"), false)
	if err != nil {
		t.Fatal(err)
	}
	for _, method := range []string{"POST", "PUT", "PATCH", "DELETE"} {
		r := httptest.NewRequest(method, "http://family.local/api/profiles/missing", nil)
		r.Header.Set("Origin", "https://another.example")
		w := httptest.NewRecorder()
		s.Routes().ServeHTTP(w, r)
		if w.Code != http.StatusForbidden {
			t.Errorf("%s got %d", method, w.Code)
		}
	}
}

func TestClientLockSurvivesRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state")
	s, err := New(path, false)
	if err != nil {
		t.Fatal(err)
	}
	send := func(path, client, password string) int {
		r := httptest.NewRequest("POST", path, strings.NewReader(`{"password":"`+password+`"}`))
		r.RemoteAddr = client
		w := httptest.NewRecorder()
		s.Routes().ServeHTTP(w, r)
		return w.Code
	}
	password := "correct horse battery staple"
	if send("/api/setup", "192.0.2.1:4000", password) != 201 {
		t.Fatal("setup")
	}
	for i := 0; i < 5; i++ {
		if send("/api/login", "192.0.2.2:5000", "wrong password") != 401 {
			t.Fatal("failed attempt")
		}
	}
	s, err = New(path, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := send("/api/login", "192.0.2.2:5001", password); got != 429 {
		t.Errorf("restart lost lock: %d", got)
	}
	s.data.Sessions["expired"] = time.Now().Add(-time.Hour).Unix()
	if got := send("/api/login", "192.0.2.3:5000", password); got != 200 {
		t.Errorf("independent client: %d", got)
	}
	if _, ok := s.data.Sessions["expired"]; ok {
		t.Error("stale session retained")
	}
}

func TestOptionalParentPINSetupAndRemoval(t *testing.T) {
	path := filepath.Join(t.TempDir(), "household.json")
	s, err := New(path, false)
	if err != nil {
		t.Fatal(err)
	}
	send := func(method, path, body string, parent *http.Cookie) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if parent != nil {
			r.AddCookie(parent)
		}
		w := httptest.NewRecorder()
		s.Routes().ServeHTTP(w, r)
		return w
	}
	passwordOnly := `{"password":"correct horse battery staple"}`
	credentials := `{"password":"correct horse battery staple","pin":"001234"}`
	if send("POST", "/api/setup", credentials, nil).Code != 201 {
		t.Fatal("setup with PIN")
	}
	s, err = New(path, false)
	if err != nil {
		t.Fatal(err)
	}
	if send("POST", "/api/login", passwordOnly, nil).Code != 401 {
		t.Fatal("missing PIN accepted")
	}
	w := send("POST", "/api/login", credentials, nil)
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	first := w.Result().Cookies()[0]
	other := send("POST", "/api/login", credentials, nil).Result().Cookies()[0]
	if send("PUT", "/api/admin/pin", `{}`, first).Code != 400 || send("PUT", "/api/admin/pin", `{"pin":"123a56"}`, first).Code != 400 {
		t.Fatal("invalid change")
	}
	if send("PUT", "/api/admin/pin", `{"pin":""}`, first).Code != 200 {
		t.Fatal("remove PIN")
	}
	if send("GET", "/api/admin", "", other).Code != 401 || send("GET", "/api/admin", "", first).Code != 200 || send("POST", "/api/login", passwordOnly, nil).Code != 200 {
		t.Fatal("PIN removal or parent session rotation")
	}
}
