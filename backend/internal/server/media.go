package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Media struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Path     string  `json:"path"`
	Duration float64 `json:"duration_seconds"`
	Size     int64   `json:"size"`
	Modified int64   `json:"modified_ns"`
	Captions bool    `json:"captions"`
}

func contentType(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp4", ".m4v":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	}
	return ""
}
func openMedia(path string) (*os.File, os.FileInfo, error) {
	dir := os.Getenv("FAMILYTV_MEDIA_DIR")
	if dir == "" || !filepath.IsLocal(path) || filepath.ToSlash(filepath.Clean(path)) != path {
		return nil, nil, errors.New("invalid media location")
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, nil, err
	}
	defer root.Close()
	info, err := root.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil, nil, errors.New("file unavailable")
	}
	file, err := root.Open(path)
	if err != nil {
		return nil, nil, err
	}
	info, err = file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		file.Close()
		return nil, nil, errors.New("file unavailable")
	}
	return file, info, nil
}
func probeMedia(ctx context.Context, file *os.File) (float64, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	// Probe the already-open file descriptor. Permit only the supported local containers.
	cmd := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-protocol_whitelist", "file", "-format_whitelist", "mov,matroska,webm", "-select_streams", "v:0", "-show_entries", "stream=codec_type:format=duration", "-of", "json", "/proc/self/fd/3")
	cmd.ExtraFiles = []*os.File{file}
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	var result struct {
		Format struct {
			Duration float64 `json:"duration,string"`
		} `json:"format"`
		Streams []struct {
			Codec string `json:"codec_type"`
		} `json:"streams"`
	}
	if err = json.Unmarshal(output, &result); err != nil || len(result.Streams) == 0 || result.Streams[0].Codec != "video" || result.Format.Duration <= 0 || math.IsInf(result.Format.Duration, 0) || math.IsNaN(result.Format.Duration) {
		return 0, errors.New("no playable video duration")
	}
	return result.Format.Duration, nil
}
func captionPath(path string) string { return strings.TrimSuffix(path, filepath.Ext(path)) + ".vtt" }
func openCaption(path string) (*os.File, os.FileInfo, error) {
	f, info, err := openMedia(captionPath(path))
	if err != nil {
		return nil, nil, err
	}
	var prefix [7]byte
	n, _ := f.ReadAt(prefix[:], 0)
	if n < 7 || string(prefix[:6]) != "WEBVTT" || !strings.ContainsRune("\t \r\n", rune(prefix[6])) || info.Size() > 4<<20 {
		f.Close()
		return nil, nil, errors.New("invalid WebVTT")
	}
	return f, info, nil
}
func (s *Server) mediaList(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.authorized(r) {
		write(w, 401, map[string]string{"error": "authentication required"})
		return
	}
	write(w, 200, map[string]any{"media": append([]Media{}, s.data.Media...)})
}
func (s *Server) importMedia(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.authorized(r) {
		write(w, 401, map[string]string{"error": "authentication required"})
		return
	}
	var input struct {
		Path  string `json:"path"`
		Title string `json:"title"`
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192))
	dec.DisallowUnknownFields()
	if dec.Decode(&input) != nil || dec.Decode(&struct{}{}) != io.EOF || contentType(input.Path) == "" {
		write(w, 400, map[string]string{"error": "Choose an MP4 or WebM file."})
		return
	}
	input.Title = strings.TrimSpace(input.Title)
	if input.Title == "" {
		input.Title = strings.TrimSuffix(filepath.Base(input.Path), filepath.Ext(input.Path))
	}
	if len([]rune(input.Title)) > 160 {
		write(w, 400, map[string]string{"error": "Use a title of up to 160 characters."})
		return
	}
	for _, m := range s.data.Media {
		if m.Path == input.Path {
			write(w, 409, map[string]string{"error": "Already imported."})
			return
		}
	}
	f, info, err := openMedia(input.Path)
	if err != nil {
		write(w, 400, map[string]string{"error": "File unavailable in the configured media folder."})
		return
	}
	defer f.Close()
	duration, err := probeMedia(r.Context(), f)
	if err != nil {
		write(w, 422, map[string]string{"error": "Could not read a video track and duration. Check the file and FFprobe installation."})
		return
	}
	m := Media{ID: token(), Title: input.Title, Path: input.Path, Duration: duration, Size: info.Size(), Modified: info.ModTime().UnixNano()}
	if f, _, err := openCaption(m.Path); err == nil {
		f.Close()
		m.Captions = true
	}
	s.data.Media = append(s.data.Media, m)
	if err = s.persist(); err != nil {
		s.data.Media = s.data.Media[:len(s.data.Media)-1]
		write(w, 500, map[string]string{"error": "storage failure"})
		return
	}
	write(w, 201, m)
}
func (s *Server) parentMediaFile(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	if !s.authorized(r) {
		s.mu.Unlock()
		write(w, 401, map[string]string{"error": "authentication required"})
		return
	}
	var media Media
	for _, m := range s.data.Media {
		if m.ID == r.PathValue("id") {
			media = m
			break
		}
	}
	s.mu.Unlock()
	if media.ID == "" {
		http.NotFound(w, r)
		return
	}
	f, info, err := openMedia(media.Path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	if info.Size() != media.Size || info.ModTime().UnixNano() != media.Modified {
		write(w, 409, map[string]string{"error": "The source file changed."})
		return
	}
	mime := contentType(media.Path)
	if strings.HasSuffix(r.URL.Path, "/captions") {
		if !media.Captions {
			http.NotFound(w, r)
			return
		}
		cap, capInfo, err := openCaption(media.Path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer cap.Close()
		f, info, mime = cap, capInfo, "text/vtt; charset=utf-8"
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "private, no-store")
	http.ServeContent(w, r, "", info.ModTime(), f)
}

// Removing the catalog entry leaves the household's read-only source files in place.
func (s *Server) removeMedia(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.authorized(r) {
		write(w, 401, map[string]string{"error": "authentication required"})
		return
	}
	for i, item := range s.data.Media {
		if item.ID != r.PathValue("id") {
			continue
		}
		previous := s.data.Media
		s.data.Media = append(append([]Media(nil), previous[:i]...), previous[i+1:]...)
		if err := s.persist(); err != nil {
			s.data.Media = previous
			write(w, 500, map[string]string{"error": "storage failure"})
			return
		}
		write(w, 200, map[string]bool{"removed": true})
		return
	}
	http.NotFound(w, r)
}
