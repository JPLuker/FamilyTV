package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Parents may supply one local track. It is fetched only by viewers who opted in.
func (s *Server) endingMusic(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	_, p, ok := s.viewer(r)
	s.mu.Unlock()
	if !ok || !p.EndingMusic {
		http.NotFound(w, r)
		return
	}
	path := os.Getenv("FAMILYTV_END_MUSIC_PATH")
	mime := map[string]string{".mp3": "audio/mpeg", ".ogg": "audio/ogg"}[strings.ToLower(filepath.Ext(path))]
	if mime == "" {
		http.NotFound(w, r)
		return
	}
	f, err := os.Open(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > 2<<20 {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "private, no-store")
	http.ServeContent(w, r, "", info.ModTime(), f)
}
