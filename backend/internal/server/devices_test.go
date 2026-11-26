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
	// Household setup can precede creation of the first child.
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
	const devices = "/api/devices"
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
	s.data.Profiles = []ChildProfile{{ID:"child"}}
	var list struct{ Devices []struct{ ID string } }
	if err := json.Unmarshal(call("GET",devices,"",parent,200).Body.Bytes(), &list); err != nil || len(list.Devices)!=1 { t.Fatal("expected one account device",err) }
	assign := `{"device":"`+list.Devices[0].ID+`","profile":"child"}`
	call("PATCH", devices, assign, child, 401)
	call("POST", session, `{"mode":"manual"}`, parent, 201)
	call("GET","/api/watch","",child,403)
	call("PATCH",devices,assign,parent,200)
	call("GET", "/api/watch", "", child, 200)
	call("DELETE", session, "", parent, 200)
	call("GET", "/api/watch", "", child, 403)
	s, err = New(s.path, false)
	if err != nil {
		t.Fatal(err)
	}
	call("POST", session, `{"mode":"manual"}`, parent, 201)
	call("GET", "/api/watch", "", child, 200)
	call("DELETE", "/api/profiles/child", "", parent, 200)
	if got:=call("GET","/api/watch/device","",child,200).Body.String(); !strings.Contains(got,`"paired":true`) {t.Fatal(got)}
	call("DELETE", devices, "", parent, 200)
	call("GET", "/api/watch", "", child, 403)
}

func TestLegacyDevicesMoveToHouseholdOnce(t *testing.T) {
	var st state
	err := json.Unmarshal([]byte(`{"devices":{"oldchild":{"screens":[{"hash":"legacy","until":2000000000}]},"":{"screens":[{"hash":"new","until":2000000000}]}}}`), &st)
	if err != nil { t.Fatal(err) }
	for i:=0; i<2; i++ {
		screens := st.Devices[householdDevices].Screens
		if len(st.Devices)!=1 || len(screens)!=2 || screens[1].Hash!="legacy" || screens[1].Profile!="oldchild" || screens[0].Profile!="" {
			t.Fatal("incorrect migration",st.Devices)
		}
		data,err:=json.Marshal(st)
		if err!=nil{t.Fatal(err)}
		if err=json.Unmarshal(data,&st);err!=nil{t.Fatal(err)}
	}
}
