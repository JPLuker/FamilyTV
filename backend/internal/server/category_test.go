package server

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanBackfillsMetadataAndPreservesParentReview(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FAMILYTV_MEDIA_DIR", root)
	if err := os.WriteFile(filepath.Join(root, "clip.info.json"), []byte(`{"title":"Drawing a bird","categories":["Art"]}`), 0600); err != nil {
		t.Fatal(err)
	}
	metadata := detectMetadata("clip.mp4")
	s, err := New(filepath.Join(t.TempDir(), "state.json"), false)
	if err != nil {
		t.Fatal(err)
	}
	s.data.Media = []Media{{Path: "clip.mp4", Review: Review{Subject: "Parent topic"}}}
	if _, err := s.storeScanItem(Media{Path: "clip.mp4", Original: metadata}, true); err != nil {
		t.Fatal(err)
	}
	m := s.data.Media[0]
	if m.Review.Category != "Art" || m.Review.Subject != "Parent topic" || len(m.Review.Profiles) != 0 {
		t.Fatal(m.Review)
	}
	if suggestCategory(Metadata{Category: "Education", Creator: "Cosmic Kids Yoga"}) != "Movement" {
		t.Fatal("creator fallback")
	}
}
