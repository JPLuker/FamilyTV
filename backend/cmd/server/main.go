package main

import (
	"github.com/JPLuker/FamilyTV/backend/internal/server"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	path := os.Getenv("FAMILYTV_STATE_PATH")
	if path == "" {
		path = "./data/state.json"
	}
	s, err := server.New(path, os.Getenv("FAMILYTV_SECURE_COOKIES") == "true")
	if err != nil {
		log.Fatal(err)
	}
	addr := os.Getenv("FAMILYTV_LISTEN")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}
	srv := http.Server{Addr: addr, Handler: s.Routes(), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second, ReadTimeout: 30 * time.Second}
	log.Fatal(srv.ListenAndServe())
}
