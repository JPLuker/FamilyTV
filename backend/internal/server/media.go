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

type MediaStream struct {
	Type  string `json:"codec_type"`
	Codec string `json:"codec_name"`
}
type Media struct {
	Streams     []MediaStream `json:"streams,omitempty"`
	Metadata    Metadata      `json:"metadata"`
	Original    Metadata      `json:"original"`
	Fingerprint string        `json:"fingerprint,omitempty"`
	Review      Review        `json:"review"`
	ID          string        `json:"id"`
	Title       string        `json:"title"`
	Path        string        `json:"path"`
	Duration    float64       `json:"duration_seconds"`
	Size        int64         `json:"size"`
	Modified    int64         `json:"modified_ns"`
	Captions    bool          `json:"captions"`
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
func probeMedia(ctx context.Context, file *os.File) (float64, []MediaStream, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	// Probe the already-open file descriptor. Permit only the supported local containers.
	cmd := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-protocol_whitelist", "file", "-format_whitelist", "mov,matroska,webm", "-show_entries", "stream=codec_type,codec_name:stream_disposition=attached_pic:format=duration", "-of", "json", "/proc/self/fd/3")
	cmd.ExtraFiles = []*os.File{file}
	output, err := cmd.Output()
	if err != nil {
		return 0, nil, err
	}
	var result struct {
		Format struct {
			Duration float64 `json:"duration,string"`
		} `json:"format"`
		Streams []struct {
			MediaStream
			Disposition struct {
				Attached int `json:"attached_pic"`
			} `json:"disposition"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return 0, nil, err
	}
	var streams []MediaStream
	video := false
	for _, track := range result.Streams {
		if track.Disposition.Attached != 0 {
			continue
		}
		if track.Type == "video" {
			video = true
		}
		streams = append(streams, track.MediaStream)
	}
	duration := result.Format.Duration
	if !video || duration <= 0 || math.IsInf(duration, 0) || math.IsNaN(duration) {
		return 0, nil, errors.New("no playable video duration")
	}
	return duration, streams, nil
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
	write(w, 200, map[string]any{"media": append([]Media{}, s.data.Media...), "gaps": missingEpisodes(s.data.Media)})
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
	fp, err := fingerprint(f, info.Size())
	if err != nil {
		write(w, 422, map[string]string{"error": "Cannot fingerprint video."})
		return
	}
	for _, existing := range s.data.Media {
		if existing.Size != info.Size() {
			continue
		}
		if existing.Fingerprint != "" && existing.Fingerprint != fp {
			continue
		}
		other, oldInfo, e := openMedia(existing.Path)
		if e != nil {
			continue
		}
		if oldInfo.Size() != info.Size() {
			other.Close()
			continue
		}
		same, e := sameBytes(f, other, info.Size())
		other.Close()
		if e != nil {
			write(w, 422, map[string]string{"error": "Cannot compare source files."})
			return
		}
		if same {
			write(w, 409, map[string]string{"error": "This video is already imported as " + existing.Title})
			return
		}
	}
	original := detectMetadata(input.Path)
	metadata := original
	if input.Title != "" {
		metadata.Title = input.Title
		metadata.Source = "parent override"
	}
	duration, streams, err := probeMedia(r.Context(), f)
	if err != nil {
		write(w, 422, map[string]string{"error": "Could not read a video track and duration. Check the file and FFprobe installation."})
		return
	}
	m := Media{Streams: streams, ID: token(), Title: metadata.Title, Metadata: metadata, Original: original, Fingerprint: fp, Path: input.Path, Duration: duration, Size: info.Size(), Modified: info.ModTime().UnixNano()}
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
	serveMedia(w, r, media, nil)
}
func (s *Server) playable(r *http.Request) (Media, int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, p, ok := s.viewer(r)
	var media Media
	if ok && v.Channel != "" {
		g := s.currentGuide(p, v, time.Now().UnixMilli())
		if g != nil && g.Now != nil {
			v.Current = g.Now.Item.ID
			v.Deadline = g.Now.End
		}
	}
	if ok && v.Current == r.PathValue("id") {
		for _, m := range s.data.Media {
			if m.ID == v.Current && available(m, p) {
				media = m
				break
			}
		}
	}
	return media, v.Deadline
}
func (s *Server) viewerMediaFile(w http.ResponseWriter, r *http.Request) {
	media, deadline := s.playable(r)
	if media.ID == "" {
		http.NotFound(w, r)
		return
	}
	if deadline != 0 {
		ctrl := http.NewResponseController(w)
		_ = ctrl.SetWriteDeadline(time.UnixMilli(deadline))
		defer ctrl.SetWriteDeadline(time.Time{})
	}
	check := func() bool {
		if r.Context().Err() != nil {
			return false
		}
		latest := s
		if s.currentState != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			state, err := s.currentState(ctx)
			if err != nil {
				return false
			}
			latest = &Server{data: state}
		}
		item, _ := latest.playable(r)
		return item.ID == media.ID
	}
	serveMedia(w, r, media, check)
}

type checkedStream struct {
	io.ReadSeeker
	check func() bool
	next  time.Time
}

func (s *checkedStream) Read(b []byte) (int, error) {
	if time.Now().After(s.next) {
		if !s.check() {
			return 0, errors.New("viewing session ended")
		}
		s.next = time.Now().Add(250 * time.Millisecond)
	}
	return s.ReadSeeker.Read(b)
}
func serveMedia(w http.ResponseWriter, r *http.Request, media Media, check func() bool) {
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
	var stream io.ReadSeeker = f
	if check != nil {
		stream = &checkedStream{ReadSeeker: f, check: check}
	}
	http.ServeContent(w, r, "", info.ModTime(), stream)
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

// Refresh legacy records without changing their approval, identity or parent overrides.
func (s *Server) refreshProbe(w http.ResponseWriter, r *http.Request) {
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
		f, info, err := openMedia(item.Path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()
		if info.Size() != item.Size || info.ModTime().UnixNano() != item.Modified {
			write(w, 409, map[string]string{"error": "The source file changed."})
			return
		}
		duration, streams, err := probeMedia(r.Context(), f)
		if err != nil {
			write(w, 422, map[string]string{"error": "Could not probe this file."})
			return
		}
		updated := item
		updated.Duration = duration
		updated.Streams = streams
		s.data.Media[i] = updated
		if err := s.persist(); err != nil {
			s.data.Media[i] = item
			write(w, 500, map[string]string{"error": "storage failure"})
			return
		}
		write(w, 200, updated)
		return
	}
	http.NotFound(w, r)
}
