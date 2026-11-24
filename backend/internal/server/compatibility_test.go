package server

import (
	"context"
	"crypto/sha256"

	"os"
	"os/exec"
	"path/filepath"

	"testing"
)

func TestCompatibleMKVRemuxAndTranscode(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is required for media fixtures")
	}
	for _, codec := range []string{"libx264", "mpeg4"} {
		t.Run(codec, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "lesson.mkv")
			cmd := exec.Command("ffmpeg", "-loglevel", "error", "-f", "lavfi", "-i", "color=s=64x64:d=1", "-f", "lavfi", "-i", "sine=frequency=220:duration=1", "-f", "lavfi", "-i", "sine=frequency=440:duration=1", "-map", "0:v", "-map", "1:a", "-map", "2:a", "-c:v", codec, "-threads", "1", "-c:a", "aac", "-disposition:a:0", "0", "-disposition:a:1", "default", path)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatal(err, string(out))
			}
			f, err := os.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			info, err := f.Stat()
			if err != nil {
				t.Fatal(err)
			}
			media := Media{ID: token(), Size: info.Size(), Modified: info.ModTime().UnixNano()}
			out, err := compatibilityFile(context.Background(), media, f, false, func() bool { return true })
			if err != nil {
				t.Fatal(err)
			}
			defer out.Close()
			duration, tracks, err := probeMedia(context.Background(), out)
			if err != nil || duration < .9 || len(tracks) != 2 || tracks[0].Codec != "h264" || tracks[1].Codec != "aac" {
				t.Fatal(duration, tracks, err)
			}
			// Compare AAC packets to identify the original default track.
			sampleHash := func(file *os.File, stream string) [32]byte {
				cmd := exec.Command("ffmpeg", "-v", "error", "-i", "/proc/self/fd/3", "-map", stream, "-c", "copy", "-f", "adts", "pipe:1")
				cmd.ExtraFiles = []*os.File{file}
				data, err := cmd.Output()
				if err != nil || len(data) == 0 {
					t.Fatal("read audio packets", err)
				}
				return sha256.Sum256(data)
			}
			if sampleHash(f, "0:a:1") != sampleHash(out, "0:a:0") {
				t.Fatal("default audio was not preserved")
			}
			if denied, err := compatibilityFile(context.Background(), media, f, false, func() bool { return false }); err == nil {
				denied.Close()
				t.Fatal("revoked viewer accessed cached output")
			}
		})
	}
}
