package server

import (
 "crypto/rand"
 "crypto/sha256"
 "crypto/hmac"
 "encoding/binary"
 "crypto/subtle"
 "encoding/base64"
 "encoding/json"
 "errors"
 "net/http"
 "net/url"
 "os"
 "path/filepath"
 "strings"
 "sync"
 "time"
)
type account struct { Salt string `json:"salt"`; Hash string `json:"hash"` }
type state struct { Account *account `json:"account,omitempty"`; Sessions map[string]int64 `json:"sessions,omitempty"` }
type Server struct { mu sync.Mutex; path string; data state; secureCookies bool; failures int; lockedUntil time.Time }
func New(path string,secure bool)(*Server,error){s:=&Server{path:path,secureCookies:secure};b,e:=os.ReadFile(path);if e==nil {if e=json.Unmarshal(b,&s.data);e!=nil{return nil,e}}else if !errors.Is(e,os.ErrNotExist){return nil,e};if s.data.Sessions==nil{s.data.Sessions=map[string]int64{}};return s,nil}
func (s *Server) persist()error{if e:=os.MkdirAll(filepath.Dir(s.path),0700);e!=nil{return e};b,e:=json.Marshal(s.data);if e!=nil{return e};f,e:=os.CreateTemp(filepath.Dir(s.path),".familytv-*");if e!=nil{return e};defer os.Remove(f.Name());if e=f.Chmod(0600);e!=nil{f.Close();return e};if _,e=f.Write(b);e!=nil{f.Close();return e};if e=f.Sync();e!=nil{f.Close();return e};if e=f.Close();e!=nil{return e};return os.Rename(f.Name(),s.path)}
func token()string{b:=make([]byte,32);if _,e:=rand.Read(b);e!=nil{panic(e)};return base64.RawURLEncoding.EncodeToString(b)}
func digest(pass,salt string)string{
 key:=[]byte(pass);sb:=[]byte(salt);block:=make([]byte,len(sb)+4);copy(block,sb);binary.BigEndian.PutUint32(block[len(sb):],1)
 mac:=hmac.New(sha256.New,key);mac.Write(block);u:=mac.Sum(nil);out:=append([]byte(nil),u...)
 for i:=1;i<310000;i++ {mac.Reset();mac.Write(u);u=mac.Sum(u[:0]);for j:=range out{out[j]^=u[j]}}
 return base64.RawStdEncoding.EncodeToString(out)
}
func write(w http.ResponseWriter,code int,data any){w.Header().Set("Content-Type","application/json");w.Header().Set("Cache-Control","no-store");w.WriteHeader(code);_ = json.NewEncoder(w).Encode(data)}
func credentials(r *http.Request)(string,error){r.Body=http.MaxBytesReader(nil,r.Body,4096);var x struct{Password string `json:"password"`};if e:=json.NewDecoder(r.Body).Decode(&x);e!=nil{return "",e};if len(x.Password)<12||len(x.Password)>256{return "",errors.New("password length")};return x.Password,nil}
func (s *Server) authorized(r *http.Request)bool{c,e:=r.Cookie("familytv_admin");if e!=nil{return false};expiry,ok:=s.data.Sessions[c.Value];return ok&&expiry>time.Now().Unix()}
func (s *Server) Routes()http.Handler{
 mux:=http.NewServeMux()
 mux.HandleFunc("GET /api/health",func(w http.ResponseWriter,r *http.Request){write(w,200,map[string]string{"status":"ok"})})
 mux.HandleFunc("GET /api/setup/status",func(w http.ResponseWriter,r *http.Request){s.mu.Lock();defer s.mu.Unlock();write(w,200,map[string]bool{"needs_setup":s.data.Account==nil})})
 mux.HandleFunc("POST /api/setup",func(w http.ResponseWriter,r *http.Request){pass,e:=credentials(r);if e!=nil{write(w,400,map[string]string{"error":"invalid password"});return};s.mu.Lock();defer s.mu.Unlock();if s.data.Account!=nil{write(w,409,map[string]string{"error":"already configured"});return};salt:=token();s.data.Account=&account{Salt:salt,Hash:digest(pass,salt)};if e=s.persist();e!=nil{s.data.Account=nil;write(w,500,map[string]string{"error":"storage failure"});return};write(w,201,map[string]bool{"created":true})})
 mux.HandleFunc("POST /api/login",func(w http.ResponseWriter,r *http.Request){pass,e:=credentials(r);if e!=nil{write(w,401,map[string]string{"error":"invalid credentials"});return};s.mu.Lock();defer s.mu.Unlock();if time.Now().Before(s.lockedUntil){write(w,429,map[string]string{"error":"too many attempts"});return};if s.data.Account==nil{write(w,409,map[string]string{"error":"setup required"});return};hash:=digest(pass,s.data.Account.Salt);if subtle.ConstantTimeCompare([]byte(hash),[]byte(s.data.Account.Hash))!=1{s.failures++;if s.failures>=5{s.failures=0;s.lockedUntil=time.Now().Add(5*time.Minute)};write(w,401,map[string]string{"error":"invalid credentials"});return};s.failures=0;s.lockedUntil=time.Time{};t:=token();s.data.Sessions[t]=time.Now().Add(12*time.Hour).Unix();if e=s.persist();e!=nil{delete(s.data.Sessions,t);write(w,500,map[string]string{"error":"storage failure"});return};http.SetCookie(w,&http.Cookie{Name:"familytv_admin",Value:t,Path:"/",HttpOnly:true,Secure:s.secureCookies,SameSite:http.SameSiteStrictMode,MaxAge:43200});write(w,200,map[string]bool{"authenticated":true})})
 mux.HandleFunc("POST /api/logout",func(w http.ResponseWriter,r *http.Request){s.mu.Lock();defer s.mu.Unlock();if c,e:=r.Cookie("familytv_admin");e==nil{delete(s.data.Sessions,c.Value);if e=s.persist();e!=nil{write(w,500,map[string]string{"error":"storage failure"});return}};http.SetCookie(w,&http.Cookie{Name:"familytv_admin",Path:"/",MaxAge:-1,HttpOnly:true,Secure:s.secureCookies,SameSite:http.SameSiteStrictMode});write(w,200,map[string]bool{"authenticated":false})})
 mux.HandleFunc("GET /api/admin",func(w http.ResponseWriter,r *http.Request){s.mu.Lock();defer s.mu.Unlock();if !s.authorized(r){write(w,401,map[string]string{"error":"authentication required"});return};write(w,200,map[string]string{"service":"FamilyTV","stage":"M0"})})
 if dir:=os.Getenv("FAMILYTV_WEB_DIR");dir!=""{fs:=http.FileServer(http.Dir(dir));mux.Handle("GET /",http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){if strings.HasPrefix(r.URL.Path,"/api/"){http.NotFound(w,r);return};path:=filepath.Join(dir,filepath.Clean(r.URL.Path));if _,e:=os.Stat(path);e!=nil{r2:=r.Clone(r.Context());r2.URL.Path="/";fs.ServeHTTP(w,r2);return};fs.ServeHTTP(w,r)}))}
 return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){w.Header().Set("X-Content-Type-Options","nosniff");w.Header().Set("Referrer-Policy","no-referrer");w.Header().Set("X-Frame-Options","DENY");if r.Method=="POST"&&r.Header.Get("Origin")!=""{u,e:=url.Parse(r.Header.Get("Origin"));if e!=nil||u.Host!=r.Host||(u.Scheme!="http"&&u.Scheme!="https")||(s.secureCookies&&u.Scheme!="https"){http.Error(w,"forbidden origin",403);return}};mux.ServeHTTP(w,r)})
}
