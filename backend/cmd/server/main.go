package main

import (
	"context"
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
	secure := os.Getenv("FAMILYTV_SECURE_COOKIES") == "true"
	var handler http.Handler
	if dsn := os.Getenv("FAMILYTV_DATABASE_URL"); dsn != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		h, closeDB, err := server.OpenPostgres(ctx, dsn, path, secure)
		cancel()
		if err != nil {
			log.Fatal(err)
		}
		defer closeDB()
		handler = h
	} else {
		s, err := server.New(path, secure)
		if err != nil {
			log.Fatal(err)
		}
		s.StartLibraryScanner(context.Background())
		handler = s.Routes()
	}
	addr := os.Getenv("FAMILYTV_LISTEN")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}
	srv := http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second, ReadTimeout: 30 * time.Second}
	log.Fatal(srv.ListenAndServe())
}
