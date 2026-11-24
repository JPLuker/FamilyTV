package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBulkReviewsAreScopedAndAtomic(t *testing.T) {
	s, err := New(filepath.Join(t.TempDir(), "state"), false)
	if err != nil {
		t.Fatal(err)
	}
	s.data.Media = []Media{{ID: "1", Metadata: Metadata{Creator: "Maker", Series: "Show"}, Review: Review{Category: "Robots", Profiles: []string{"other"}}}, {ID: "2"}}
	s.data.Profiles = []ChildProfile{{ID: "child", Name: "Child"}, {ID: "other"}}
	s.data.Sessions["parent"] = time.Now().Add(time.Hour).Unix()
	call := func(admin bool, ids string) int {
		r := httptest.NewRequest("POST", "/api/approvals", strings.NewReader(`{"profile":"child","action":"approve","ids":`+ids+`}`))
		if admin {
			r.AddCookie(&http.Cookie{Name: "familytv_admin", Value: "parent"})
		}
		w := httptest.NewRecorder()
		s.Routes().ServeHTTP(w, r)
		return w.Code
	}
	if call(false, `["1"]`) != 401 || call(true, `["1","gone"]`) != 409 || s.data.Media[0].Reviewed {
		t.Fatal("unexpected changes")
	}
	s.save = func(state) error { return errors.New("offline") }
	if call(true, `["1","2"]`) != 500 || len(s.data.ApprovalLog) != 0 || len(s.data.Media[0].Review.Profiles) != 1 {
		t.Fatal("partial write")
	}
	s.save = nil
	if call(true, `["1","2"]`) != 200 || call(true, `["1","2"]`) != 200 || len(s.data.ApprovalLog) != 1 || s.data.ApprovalLog[0].Count != 2 {
		t.Fatal("bulk operation or retry")
	}
	next, err := New(s.path, false)
	if err != nil {
		t.Fatal(err)
	}
	m := next.data.Media[0]
	if !eligible(m, next.data.Profiles[0]) || !eligible(m, next.data.Profiles[1]) {
		t.Fatal("lost approval")
	}
	for _, p := range []ChildProfile{{ID: "child", BlockedCreators: " maker "}, {ID: "child", BlockedSeries: "Other\nSHOW"}, {ID: "child", BlockedCategories: "robots"}} {
		if eligible(m, p) {
			t.Fatal("group restriction bypassed", p)
		}
	}
}
