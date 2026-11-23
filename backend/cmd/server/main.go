package main
import ("log";"net/http";"os";"github.com/JPLuker/FamilyTV/backend/internal/server")
func main(){path:=os.Getenv("FAMILYTV_STATE_PATH");if path==""{path="./data/state.json"};s,err:=server.New(path,os.Getenv("FAMILYTV_SECURE_COOKIES")=="true");if err!=nil{log.Fatal(err)};addr:=os.Getenv("FAMILYTV_LISTEN");if addr==""{addr="127.0.0.1:8080"};log.Fatal(http.ListenAndServe(addr,s.Routes()))}
