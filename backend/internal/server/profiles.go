package server

import (
 "encoding/json"
 "net/http"
 "strings"
)
type ChildProfile struct {
 ID string `json:"id"`
 Name string `json:"name"`
 Captions bool `json:"captions"`
 ReducedMotion bool `json:"reduced_motion"`
 AudioDescription bool `json:"audio_description"`
 Navigation string `json:"navigation"`
}
func (s *Server) listProfiles(w http.ResponseWriter,r *http.Request){
 s.mu.Lock();defer s.mu.Unlock()
 if !s.authorized(r){write(w,401,map[string]string{"error":"authentication required"});return}
 profiles:=append([]ChildProfile{},s.data.Profiles...)
 write(w,200,map[string]any{"profiles":profiles})
}
func (s *Server) createProfile(w http.ResponseWriter,r *http.Request){
 s.mu.Lock();defer s.mu.Unlock()
 if !s.authorized(r){write(w,401,map[string]string{"error":"authentication required"});return}
 r.Body=http.MaxBytesReader(w,r.Body,4096)
 var p ChildProfile
 dec:=json.NewDecoder(r.Body);dec.DisallowUnknownFields()
 if err:=dec.Decode(&p);err!=nil{write(w,400,map[string]string{"error":"invalid profile"});return}
 p.Name=strings.TrimSpace(p.Name)
 if len(p.Name)<1||len([]rune(p.Name))>50||p.ID!=""||len(s.data.Profiles)>=20{write(w,400,map[string]string{"error":"invalid profile"});return}
 if p.Navigation==""{p.Navigation="visual"}
 if p.Navigation!="visual"&&p.Navigation!="visual-text"&&p.Navigation!="text"{write(w,400,map[string]string{"error":"invalid navigation mode"});return}
 p.ID=token()
 s.data.Profiles=append(s.data.Profiles,p)
 if err:=s.persist();err!=nil{s.data.Profiles=s.data.Profiles[:len(s.data.Profiles)-1];write(w,500,map[string]string{"error":"storage failure"});return}
 write(w,201,p)
}
func (s *Server) deleteProfile(w http.ResponseWriter,r *http.Request){
 s.mu.Lock();defer s.mu.Unlock()
 if !s.authorized(r){write(w,401,map[string]string{"error":"authentication required"});return}
 id:=r.PathValue("id")
 for i,p:=range s.data.Profiles{if p.ID==id{
  before:=append([]ChildProfile(nil),s.data.Profiles...)
  s.data.Profiles=append(s.data.Profiles[:i],s.data.Profiles[i+1:]...)
  if err:=s.persist();err!=nil{s.data.Profiles=before;write(w,500,map[string]string{"error":"storage failure"});return}
  write(w,200,map[string]bool{"deleted":true});return
 }}
 write(w,404,map[string]string{"error":"profile not found"})
}
