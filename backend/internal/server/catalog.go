package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
)

type Metadata struct {
	Attribution string `json:"attribution,omitempty"`
	Title       string `json:"title"`
	Kind        string `json:"kind"`
	Series      string `json:"series"`
	Season      int    `json:"season"`
	Episode     int    `json:"episode"`
	Creator     string `json:"creator"`
	Playlist    string `json:"playlist"`
	ExternalID  string `json:"external_id"`
	Language    string `json:"language"`
	License     string `json:"license"`
	Source      string `json:"source"`
	Collection  string `json:"collection"`
}

var episodePattern = regexp.MustCompile(`(?i)^(.*?)[ ._-]*s([0-9]{1,3})e([0-9]{1,4})(?:[ ._-]+(.*))?$`)

func sidecar(path string) ([]byte, error) {
	f, info, e := openMedia(path)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	if info.Size() > 2<<20 {
		return nil, errors.New("sidecar too large")
	}
	return io.ReadAll(io.LimitReader(f, 2<<20))
}
func detectMetadata(path string) Metadata {
	stem := strings.TrimSuffix(path, filepath.Ext(path))
	m := Metadata{Title: filepath.Base(stem), Kind: "video", Source: "filename"}
	if parts := episodePattern.FindStringSubmatch(m.Title); parts != nil {
		m.Series = strings.TrimSpace(strings.ReplaceAll(parts[1], ".", " "))
		m.Season, _ = strconv.Atoi(parts[2])
		m.Episode, _ = strconv.Atoi(parts[3])
		m.Kind = "episode"
		if m.Season == 0 {
			m.Kind = "special"
		}
		if parts[4] != "" {
			m.Title = strings.ReplaceAll(parts[4], ".", " ")
		}
	}
	if b, e := sidecar(stem + ".nfo"); e == nil {
		var n struct {
			XMLName xml.Name
			Title   string `xml:"title"`
			Series  string `xml:"showtitle"`
			Season  int    `xml:"season"`
			Episode int    `xml:"episode"`
			ID      string `xml:"uniqueid"`
		}
		if xml.Unmarshal(b, &n) == nil && n.Title != "" && (n.XMLName.Local == "episodedetails" || n.XMLName.Local == "movie") {
			m.Title = n.Title
			m.Source = "local NFO"
			m.ExternalID = n.ID
			m.Series = n.Series
			m.Season = n.Season
			m.Episode = n.Episode
			m.Kind = "movie"
			if n.XMLName.Local == "episodedetails" {
				m.Kind = "episode"
				if n.Season == 0 {
					m.Kind = "special"
				}
			}
			return m
		}
	}
	if b, e := sidecar(stem + ".info.json"); e == nil {
		var y struct {
			Title, Channel, Uploader, ID, Playlist, Language, License string
			PlaylistTitle                                             string `json:"playlist_title"`
		}
		if json.Unmarshal(b, &y) == nil && y.Title != "" {
			m.Title = y.Title
			m.Kind = "creator"
			m.Creator = y.Channel
			if m.Creator == "" {
				m.Creator = y.Uploader
			}
			m.ExternalID = y.ID
			m.Playlist = y.PlaylistTitle
			if m.Playlist == "" {
				m.Playlist = y.Playlist
			}
			m.Language = y.Language
			m.License = y.License
			m.Source = "local video metadata"
		}
	}
	return m
}
func fingerprint(f *os.File, size int64) (string, error) {
	// Sample to find likely duplicates. Exact content is compared before rejecting an import.
	h := sha256.New()
	fmt.Fprint(h, size, ":")
	for _, off := range []int64{0, max(0, size/2-32768), max(0, size-65536)} {
		if _, err := io.Copy(h, io.NewSectionReader(f, off, 65536)); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
func sameBytes(a, b *os.File, size int64) (bool, error) {
	sum := func(f *os.File) (string, error) {
		h := sha256.New()
		_, e := io.Copy(h, io.NewSectionReader(f, 0, size))
		return string(h.Sum(nil)), e
	}
	first, e := sum(a)
	if e != nil {
		return false, e
	}
	second, e := sum(b)
	return first == second, e
}
func (s *Server) scanMedia(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	allowed := s.authorized(r)
	s.mu.Unlock()
	if !allowed {
		write(w, 401, map[string]string{"error": "authentication required"})
		return
	}
	root, err := os.OpenRoot(os.Getenv("FAMILYTV_MEDIA_DIR"))
	if err != nil {
		write(w, 400, map[string]string{"error": "Media folder unavailable."})
		return
	}
	defer root.Close()
	after := r.URL.Query().Get("after")
	paths := []string{}
	truncated := false
	skipped := 0
	err = fs.WalkDir(root.FS(), ".", func(path string, d fs.DirEntry, err error) error {
		if r.Context().Err() != nil {
			return r.Context().Err()
		}
		if err != nil {
			skipped++
			return nil
		}
		if d.IsDir() {
			if path != "." && strings.HasPrefix(d.Name(), ".") {
				return fs.SkipDir
			}
			return nil
		}
		if contentType(path) == "" || !d.Type().IsRegular() {
			return nil
		}
		// WalkDir traverses sibling names in order, with each subtree visited as a unit.
		if after != "" {
			if path == after {
				after = ""
			}
			return nil
		}
		if len(paths) >= 200 {
			truncated = true
			return fs.SkipAll
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		write(w, 500, map[string]string{"error": "Scan interrupted."})
		return
	}
	write(w, 200, map[string]any{"paths": paths, "more": truncated, "skipped": skipped})
}
func (s *Server) editMetadata(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.authorized(r) {
		write(w, 401, map[string]string{"error": "authentication required"})
		return
	}
	var value Metadata
	if r.Method == "PUT" {
		if !readInput(w, r, &value) {
			return
		}
		value.Title = strings.TrimSpace(value.Title)
		if value.Title == "" || len([]rune(value.Title)) > 160 || !slices.Contains([]string{"episode", "special", "movie", "creator", "video"}, value.Kind) || value.Season < 0 || value.Episode < 0 {
			write(w, 400, map[string]string{"error": "Invalid title, format or episode number."})
			return
		}
		if value.Attribution != "" && !regexp.MustCompile(`^https://(www\.tvmaze\.com/shows/[0-9]+|www\.youtube\.com/watch\?v=[A-Za-z0-9_-]{11})$`).MatchString(value.Attribution) {
			write(w, 400, map[string]string{"error": "Invalid attribution URL."})
			return
		}
		value.Source = "parent override"
	}
	for i, m := range s.data.Media {
		if m.ID == r.PathValue("id") {
			if r.Method == "DELETE" {
				value = m.Original
				if value.Title == "" {
					value = detectMetadata(m.Path)
				}
			}
			if m.Original.Title == "" {
				s.data.Media[i].Original = detectMetadata(m.Path)
			}
			s.data.Media[i].Metadata = value
			s.data.Media[i].Title = value.Title
			if err := s.persist(); err != nil {
				s.data.Media[i] = m
				write(w, 500, map[string]string{"error": "storage failure"})
				return
			}
			write(w, 200, s.data.Media[i])
			return
		}
	}
	http.NotFound(w, r)
}

type Gap struct {
	Series string `json:"series"`
	Season int    `json:"season"`
	From   int    `json:"from"`
	To     int    `json:"to"`
}

func missingEpisodes(media []Media) []Gap {
	type key struct {
		series string
		season int
	}
	seen := map[key][]int{}
	for _, m := range media {
		x := m.Metadata
		if x.Kind == "episode" && x.Series != "" && x.Episode > 0 {
			k := key{x.Series, x.Season}
			seen[k] = append(seen[k], x.Episode)
		}
	}
	gaps := []Gap{}
	for k, numbers := range seen {
		sort.Ints(numbers)
		last := 0
		for _, n := range numbers {
			if n > last+1 {
				gaps = append(gaps, Gap{k.series, k.season, last + 1, n - 1})
			}
			last = n
		}
	}
	sort.Slice(gaps, func(i, j int) bool {
		a, b := gaps[i], gaps[j]
		if a.Series != b.Series {
			return a.Series < b.Series
		}
		if a.Season != b.Season {
			return a.Season < b.Season
		}
		return a.From < b.From
	})
	return gaps
}
