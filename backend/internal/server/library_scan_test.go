package server

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRecursiveScanImportsOnceAndRequiresReview(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("requires ffmpeg")
	}
	root := t.TempDir()
	t.Setenv("FAMILYTV_MEDIA_DIR", root)
	t.Setenv("FAMILYTV_SCAN_ON_START", "false")
	sub := filepath.Join(root, "shows", "season")
	if err := os.MkdirAll(sub, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(sub, "lesson.mp4")
	if b, err := exec.Command("ffmpeg", "-loglevel", "error", "-f", "lavfi", "-i", "color=c=blue:s=32x32:d=1", "-c:v", "mpeg4", path).CombinedOutput(); err != nil {
		t.Fatal(err, string(b))
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "copy.mp4"), b, 0600); err != nil {
		t.Fatal(err)
	}
	s, err := New(filepath.Join(t.TempDir(), "state.json"), false)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.StartLibraryScanner(ctx)
	for i := 0; i < 2; i++ {
		s.scanner.status = scanResult{}
		s.scanner.run()
		if len(s.data.Media) != 1 || s.scanner.status.Failed != 0 || s.scanner.status.Skipped != i+1 {
			t.Fatal(s.scanner.status, len(s.data.Media))
		}
	}
	if m := s.data.Media[0]; m.Reviewed || len(m.Review.Profiles) != 0 {
		t.Fatal("scan granted approval")
	}
}

func TestFolderSelectionStaysInsideSharedMedia(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "Shows", "Season 1"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "Shows"), filepath.Join(dir, "linked")); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	for _, path := range []string{".", "Shows", "Shows/Season 1"} {
		if !libraryDirectory(root, path) {
			t.Errorf("rejected %q", path)
		}
	}
	for _, path := range []string{"", "..", "/", "Shows/../Shows", "linked", "linked/Season 1", ".hidden", "missing"} {
		if libraryDirectory(root, path) {
			t.Errorf("accepted %q", path)
		}
	}
}
