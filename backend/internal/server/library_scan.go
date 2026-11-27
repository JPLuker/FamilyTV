package server

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type scanResult struct {
	Finished  int64  `json:"finished"`
	Running   bool   `json:"running"`
	Imported  int    `json:"imported"`
	Skipped   int    `json:"skipped"`
	Failed    int    `json:"failed"`
	LastError string `json:"last_error,omitempty"`
}
type LibraryScanner struct {
	ctx    context.Context
	known  func(context.Context) (map[string]bool, string, error)
	insert func(context.Context, Media, bool) (int, error)
	mu     sync.Mutex
	status scanResult
}

func knownPaths(items []Media) map[string]bool {
	paths := make(map[string]bool, len(items))
	for _, m := range items {
		paths[m.Path] = m.CategoryChecked
	}
	return paths
}
func (s *Server) StartLibraryScanner(ctx context.Context) {
	scanner := &LibraryScanner{ctx: ctx}
	scanner.known = func(context.Context) (map[string]bool, string, error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		return knownPaths(s.data.Media), s.data.LibraryFolder, nil
	}
	scanner.insert = func(_ context.Context, m Media, existing bool) (int, error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.storeScanItem(m, existing)
	}
	s.scanner = scanner
	go scanner.schedule()
}
func (s *LibraryScanner) schedule() {
	if os.Getenv("FAMILYTV_SCAN_ON_START") != "false" {
		s.start()
	}
	t := time.NewTicker(time.Hour)
	defer t.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-t.C:
			s.start()
		}
	}
}
func (s *LibraryScanner) start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.startLocked()
}
func (s *LibraryScanner) startLocked() {
	if s.status.Running {
		return
	}
	s.status = scanResult{Running: true}
	go s.run()
}
func (s *LibraryScanner) run() {
	defer func() {
		s.mu.Lock()
		s.status.Running = false
		s.status.Finished = time.Now().UnixMilli()
		s.mu.Unlock()
	}()
	failed := func(message string) { s.mu.Lock(); s.status.Failed++; s.status.LastError = message; s.mu.Unlock() }
	known, folder, err := s.known(s.ctx)
	if err != nil {
		failed("Library unavailable.")
		return
	}
	root, err := os.OpenRoot(os.Getenv("FAMILYTV_MEDIA_DIR"))
	if err != nil {
		failed("Media folder unavailable.")
		return
	}
	defer root.Close()
	folder = defaultFolder(folder)
	if !libraryDirectory(root, folder) {
		failed("Selected folder unavailable.")
		return
	}
	err = fs.WalkDir(root.FS(), folder, func(path string, d fs.DirEntry, err error) error {
		if s.ctx.Err() != nil {
			return s.ctx.Err()
		}
		if err != nil {
			failed("Cannot read " + path)
			return nil
		}
		if d.IsDir() {
			if path != "." && strings.HasPrefix(d.Name(), ".") {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() || contentType(path) == "" {
			return nil
		}
		status := 409
		if checked, exists := known[path]; exists && !checked {
			info := detectMetadata(path)
			status, err = s.insert(s.ctx, Media{Path: path, Original: info}, true)
		} else if !exists {
			var m Media
			m, status, err = prepareMedia(s.ctx, path, "")
			if err == nil {
				status, err = s.insert(s.ctx, m, false)
			}
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		switch status {
		case 201:
			s.status.Imported++
		case 409:
			s.status.Skipped++
		default:
			s.status.Failed++
			s.status.LastError = "Could not import " + path
			if err != nil {
				s.status.LastError += ": " + err.Error()
			}
		}
		return nil
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		failed("Scan interrupted.")
	}
}
func (s *Server) libraryScan(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	authorized := s.authorized(r)
	s.mu.Unlock()
	if !authorized {
		write(w, 401, map[string]string{"error": "authentication required"})
		return
	}
	scanner := s.scanner
	if scanner == nil {
		write(w, 503, map[string]string{"error": "Scanner unavailable."})
		return
	}
	if r.Method == "POST" {
		scanner.start()
	}
	scanner.mu.Lock()
	defer scanner.mu.Unlock()
	write(w, 200, scanner.status)
}
