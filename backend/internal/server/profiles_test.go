package server
import ("bytes";"encoding/json";"net/http";"net/http/httptest";"path/filepath";"testing")
func TestProfileRequiresParentAndPersists(t *testing.T){
 path:=filepath.Join(t.TempDir(),"state.json")
 s,e:=New(path,false);if e!=nil{t.Fatal(e)}
 call:=func(s *Server,method,url,body string,c *http.Cookie)*httptest.ResponseRecorder{
  r:=httptest.NewRequest(method,"http://localhost"+url,bytes.NewBufferString(body));if c!=nil{r.AddCookie(c)}
  w:=httptest.NewRecorder();s.Routes().ServeHTTP(w,r);return w
 }
 if w:=call(s,"POST","/api/profiles",`{"name":"Sam"}`,nil);w.Code!=401{t.Fatal(w.Code)}
 if w:=call(s,"POST","/api/setup",`{"password":"correct horse battery staple"}`,nil);w.Code!=201{t.Fatal(w.Code)}
 w:=call(s,"POST","/api/login",`{"password":"correct horse battery staple"}`,nil)
 if w.Code!=200{t.Fatal(w.Code)}
 cookie:=w.Result().Cookies()[0]
 w=call(s,"POST","/api/profiles",`{"name":"Sam","captions":true,"navigation":"visual-text"}`,cookie)
 if w.Code!=201{t.Fatal(w.Code,w.Body.String())}
 var profile ChildProfile
 if err:=json.Unmarshal(w.Body.Bytes(),&profile);err!=nil{t.Fatal(err)}
 if !profile.Captions||profile.ID==""{t.Fatal(profile)}
 s2,e:=New(path,false);if e!=nil{t.Fatal(e)}
 w=call(s2,"GET","/api/profiles","",cookie)
 if w.Code!=200||!bytes.Contains(w.Body.Bytes(),[]byte("Sam")){t.Fatal(w.Code,w.Body.String())}
 w=call(s2,"DELETE","/api/profiles/"+profile.ID,"",cookie)
 if w.Code!=200{t.Fatal(w.Code,w.Body.String())}
 w=call(s2,"GET","/api/profiles","",cookie)
 if bytes.Contains(w.Body.Bytes(),[]byte("Sam")){t.Fatal("deleted profile still present")}
}
