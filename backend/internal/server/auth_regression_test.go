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
