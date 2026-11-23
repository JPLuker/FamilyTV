package server
import ("bytes";"net/http";"net/http/httptest";"path/filepath";"testing")
func TestParentSetupLoginAndLogout(t *testing.T){
 s,e:=New(filepath.Join(t.TempDir(),"state.json"),false);if e!=nil{t.Fatal(e)};h:=s.Routes()
 send:=func(method,path,body string,cookie *http.Cookie)*httptest.ResponseRecorder{r:=httptest.NewRequest(method,"http://family.local"+path,bytes.NewBufferString(body));if cookie!=nil{r.AddCookie(cookie)};w:=httptest.NewRecorder();h.ServeHTTP(w,r);return w}
 if w:=send("GET","/api/admin","",nil);w.Code!=401{t.Fatal(w.Code)}
 if w:=send("POST","/api/setup",`{"password":"correct horse battery staple"}`,nil);w.Code!=201{t.Fatal(w.Code)}
 if w:=send("POST","/api/setup",`{"password":"another passphrase"}`,nil);w.Code!=409{t.Fatal(w.Code)}
 if w:=send("POST","/api/login",`{"password":"incorrect password"}`,nil);w.Code!=401{t.Fatal(w.Code)}
 w:=send("POST","/api/login",`{"password":"correct horse battery staple"}`,nil);if w.Code!=200{t.Fatal(w.Code)}
 cookies:=w.Result().Cookies();if len(cookies)!=1||!cookies[0].HttpOnly{t.Fatal("missing HttpOnly cookie")}
 if w:=send("GET","/api/admin","",cookies[0]);w.Code!=200{t.Fatal(w.Code)}
 if w:=send("POST","/api/logout","",cookies[0]);w.Code!=200{t.Fatal(w.Code)}
 if w:=send("GET","/api/admin","",cookies[0]);w.Code!=401{t.Fatal(w.Code)}
}
func TestForgedOriginRejected(t *testing.T){
 s,_:=New(filepath.Join(t.TempDir(),"state.json"),false)
 for _,origin:=range []string{"https://family.local.evil.example","https://evil.example/?next=://family.local","null"}{
  r:=httptest.NewRequest("POST","http://family.local/api/setup",bytes.NewBufferString(`{"password":"correct horse battery staple"}`));r.Header.Set("Origin",origin);w:=httptest.NewRecorder();s.Routes().ServeHTTP(w,r);if w.Code!=403{t.Fatalf("%s: %d",origin,w.Code)}
 }
}
func TestLoginLockout(t *testing.T){
 s,_:=New(filepath.Join(t.TempDir(),"state.json"),false);h:=s.Routes()
 send:=func(path,pass string)int{r:=httptest.NewRequest("POST",path,bytes.NewBufferString(`{"password":"`+pass+`"}`));w:=httptest.NewRecorder();h.ServeHTTP(w,r);return w.Code}
 if got:=send("/api/setup","correct horse battery staple");got!=201{t.Fatal(got)}
 for i:=0;i<5;i++{if got:=send("/api/login","incorrect password");got!=401{t.Fatal(got)}}
 if got:=send("/api/login","correct horse battery staple");got!=429{t.Fatal(got)}
}
