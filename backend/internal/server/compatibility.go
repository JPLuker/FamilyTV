package server

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"
)

// Keep one seekable compatibility copy. Independent handles remain valid across eviction.
// The anonymous file disappears when its last handle closes, including on process exit.
var conversionSlot = make(chan struct{}, 1)
var cachedConversion struct {
	sync.Mutex
	key  string
	file *os.File
}

const maxConversionBytes = 512 << 20

func openCachedConversion(key string) (*os.File, error) {
	cachedConversion.Lock()
	defer cachedConversion.Unlock()
	if cachedConversion.file == nil || cachedConversion.key != key {
		return nil, nil
	}
	return os.Open("/proc/self/fd/" + strconv.Itoa(int(cachedConversion.file.Fd())))
}

func compatibilityFile(ctx context.Context, m Media, input *os.File, force bool, allowed func() bool) (*os.File, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()
	key := fmt.Sprintf("%s:%d:%d:%t", m.ID, m.Size, m.Modified, force)
	if !allowed() {
		return nil, errors.New("session ended")
	}
	if file, err := openCachedConversion(key); file != nil || err != nil {
		return file, err
	}
	select {
	case conversionSlot <- struct{}{}:
		defer func() { <-conversionSlot }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if !allowed() {
		return nil, errors.New("session ended")
	}
	if file, err := openCachedConversion(key); file != nil || err != nil {
		return file, err
	}

	output, err := os.CreateTemp("", "familytv-play-*.mp4")
	if err != nil {
		return nil, err
	}
	if err = os.Remove(output.Name()); err != nil {
		output.Close()
		return nil, err
	}
	// Cancel preparation if the parent stops playback or changes eligibility.
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if !allowed() {
					cancel()
					return
				}
			}
		}
	}()
	err = convertMP4(ctx, input, output, force)
	cancel()
	<-done
	if err != nil || !allowed() {
		output.Close()
		return nil, errors.New("conversion interrupted")
	}
	cachedConversion.Lock()
	if cachedConversion.file != nil {
		cachedConversion.file.Close()
	}
	cachedConversion.file = output
	cachedConversion.key = key
	cachedConversion.Unlock()
	return openCachedConversion(key)
}

func convertMP4(ctx context.Context, input, output *os.File, force bool) error {
	sourceDuration, tracks, err := probeMedia(ctx, input)
	if err != nil {
		return err
	}
	video, audio := "", ""
	audioIndex, selectedAudio := 0, 0
	for _, track := range tracks {
		switch track.Type {
		case "video":
			if video == "" {
				video = track.Codec
			}
		case "audio":
			if audio == "" || track.Default {
				audio = track.Codec
				selectedAudio = audioIndex
			}
			audioIndex++
		}
	}
	args := []string{"-nostdin", "-v", "error", "-y", "-protocol_whitelist", "file", "-format_whitelist", "mov,matroska,webm", "-threads", "2", "-i", "/proc/self/fd/3", "-map", "0:V:0", "-map", fmt.Sprintf("0:a:%d?", selectedAudio), "-map_metadata", "-1", "-map_chapters", "-1"}
	if video == "h264" && !force {
		args = append(args, "-c:v", "copy")
	} else {
		args = append(args, "-vf", "scale=trunc(iw/2)*2:trunc(ih/2)*2", "-c:v", "libx264", "-threads", "2", "-preset", "veryfast", "-pix_fmt", "yuv420p", "-crf", "23")
	}
	if audio == "aac" && !force {
		args = append(args, "-c:a", "copy")
	} else {
		args = append(args, "-c:a", "aac", "-ac", "2", "-b:a", "128k")
	}
	args = append(args, "-fs", strconv.Itoa(maxConversionBytes), "-f", "mp4", "/proc/self/fd/4")
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	cmd.ExtraFiles = []*os.File{input, output}
	if err := cmd.Run(); err != nil {
		return err
	}
	info, err := output.Stat()
	if err != nil {
		return err
	}
	if info.Size() == 0 || info.Size() >= maxConversionBytes {
		return errors.New("compatibility copy outside size limit")
	}
	duration, _, err := probeMedia(ctx, output)
	if err != nil || duration+0.5 < sourceDuration {
		return errors.New("incomplete compatibility copy")
	}
	return nil
}
