package server

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPostgresImportAndConcurrentUpdates(t *testing.T) {
	dsn := os.Getenv("FAMILYTV_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("requires PostgreSQL test database")
	}
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	admin, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := "familytv_test_" + strings.ToLower(token()[:12])
	schema = strings.ReplaceAll(schema, "-", "_")
	if _, err = admin.Exec("CREATE SCHEMA " + schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.Exec("DROP SCHEMA " + schema + " CASCADE"); err != nil {
			t.Error(err)
		}
	}()
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	path := filepath.Join(t.TempDir(), "old.json")
	old, err := New(path, false)
	if err != nil {
		t.Fatal(err)
	}
	old.data.Sessions["existing"] = time.Now().Add(time.Hour).Unix()
	old.data.Profiles = []ChildProfile{{ID: "old", Name: "Original"}}
	if err = old.persist(); err != nil {
		t.Fatal(err)
	}
	h, closeDB, err := OpenPostgres(context.Background(), u.String(), path, false)
	if err != nil {
		t.Fatal(err)
	}
	defer closeDB()
	call := func(h http.Handler, method, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "/api/profiles", strings.NewReader(body))
		r.AddCookie(&http.Cookie{Name: "familytv_admin", Value: "existing"})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	results := make(chan int, 4)
	for i := 0; i < 4; i++ {
		go func() { results <- call(h, "POST", `{"name":"Added"}`).Code }()
	}
	for i := 0; i < 4; i++ {
		if got := <-results; got != 201 {
			t.Fatalf("creation: %d", got)
		}
	}
	if err = os.WriteFile(path, []byte("unreadable older state"), 0600); err != nil {
		t.Fatal(err)
	}
	restarted, closeAgain, err := OpenPostgres(context.Background(), u.String(), path, false)
	if err != nil {
		t.Fatal(err)
	}
	defer closeAgain()
	w := call(restarted, "GET", "")
	if w.Code != 200 || strings.Count(w.Body.String(), `"Added"`) != 4 || !strings.Contains(w.Body.String(), "Original") {
		t.Fatal(w.Code, w.Body.String())
	}
}
