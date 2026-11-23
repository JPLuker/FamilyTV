package server

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"
)

// These frames come from the household's own files and live only in bounded memory.
var pictures = struct {
	sync.Mutex
	entries map[string][]byte
	order   []string
}{entries: map[string][]byte{}}
var decoders = make(chan struct{}, 2)

func (s *Server) picture(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	v, p, ok := s.viewer(r)
	var item Media
	if ok {
		for _, m := range s.data.Media {
			if m.ID == r.PathValue("id") && fits(m, v, time.Now()) && available(m, p) {
				item = m
				break
			}
		}
	}
	s.mu.Unlock()
	if item.ID == "" {
		http.NotFound(w, r)
		return
	}
	key := item.ID
	pictures.Lock()
	frame := pictures.entries[key]
	pictures.Unlock()
	if frame == nil {
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()
		select {
		case decoders <- struct{}{}:
			defer func() { <-decoders }()
		case <-ctx.Done():
			http.Error(w, "preview unavailable", 503)
			return
		}
		f, info, err := openMedia(item.Path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()
		if info.Size() != item.Size || info.ModTime().UnixNano() != item.Modified {
			http.NotFound(w, r)
			return
		}
		cmd := exec.CommandContext(ctx, "ffmpeg", "-v", "error", "-protocol_whitelist", "file", "-format_whitelist", "mov,matroska,webm", "-threads", "1", "-i", "/proc/self/fd/3", "-an", "-sn", "-vf", "scale=240:135:force_original_aspect_ratio=decrease", "-frames:v", "1", "-threads", "1", "-c:v", "mjpeg", "-f", "image2pipe", "pipe:1")
		cmd.ExtraFiles = []*os.File{f}
		frame, err = cmd.Output()
		if err != nil || len(frame) == 0 || len(frame) > 256<<10 {
			http.NotFound(w, r)
			return
		}
		pictures.Lock()
		if _, exists := pictures.entries[key]; !exists {
			if len(pictures.order) == 48 {
				delete(pictures.entries, pictures.order[0])
				pictures.order = pictures.order[1:]
			}
			pictures.entries[key] = frame
			pictures.order = append(pictures.order, key)
		}
		pictures.Unlock()
	}
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Type", "image/jpeg")
	_, _ = w.Write(frame)
}
