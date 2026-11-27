package server

import (
	"io/fs"
	"net/http"
	"os"
	"strings"
)

func defaultFolder(path string) string {
	if path == "" {
		return "."
	}
	return path
}

// Resolve each component inside the installation's read-only media mount.
func libraryDirectory(root *os.Root, path string) bool {
	if path == "." {
		return true
	}
	if !fs.ValidPath(path) || strings.Contains(path, "\\") {
		return false
	}
	var parents []string
	for _, name := range strings.Split(path, "/") {
		if strings.HasPrefix(name, ".") {
			return false
		}
		parents = append(parents, name)
		entry, err := root.Lstat(strings.Join(parents, "/"))
		if err != nil || !entry.IsDir() {
			return false
		}
	}
	return true
}

func (s *Server) libraryFolder(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.authorized(r) {
		write(w, 401, map[string]string{"error": "Sign in as a parent."})
		return
	}
	root, err := os.OpenRoot(os.Getenv("FAMILYTV_MEDIA_DIR"))
	if err != nil {
		write(w, 503, map[string]string{"error": "FamilyTV cannot read the shared media folder."})
		return
	}
	defer root.Close()
	selected := defaultFolder(s.data.LibraryFolder)
	path := defaultFolder(r.URL.Query().Get("path"))
	if r.Method == "PUT" {
		var input struct {
			Path string `json:"path"`
		}
		if !readInput(w, r, &input) {
			return
		}
		path = input.Path
	}
	if !libraryDirectory(root, path) {
		write(w, 400, map[string]string{"error": "Choose an available folder within shared media."})
		return
	}
	entries, err := fs.ReadDir(root.FS(), path)
	if err != nil {
		write(w, 400, map[string]string{"error": "FamilyTV cannot read this folder."})
		return
	}
	folders := []string{}
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") && !strings.Contains(entry.Name(), "\\") {
			folders = append(folders, entry.Name())
		}
	}
	if r.Method == "PUT" {
		if s.scanner == nil {
			write(w, 503, map[string]string{"error": "Scanner unavailable."})
			return
		}
		s.scanner.mu.Lock()
		defer s.scanner.mu.Unlock()
		if s.scanner.status.Running {
			write(w, 409, map[string]string{"error": "A scan is running. Choose the folder again when it finishes."})
			return
		}
		previous := s.data.LibraryFolder
		s.data.LibraryFolder = path
		if err = s.persist(); err != nil {
			s.data.LibraryFolder = previous
			write(w, 500, map[string]string{"error": "Could not save the library folder."})
			return
		}
		selected = path
		s.scanner.startLocked()
	}
	write(w, 200, map[string]any{"selected": selected, "path": path, "folders": folders})
}
