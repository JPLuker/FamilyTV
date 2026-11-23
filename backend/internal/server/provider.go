package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type Match struct {
	Title      string `json:"title"`
	Creator    string `json:"creator,omitempty"`
	ExternalID string `json:"external_id"`
	Provider   string `json:"provider"`
	URL        string `json:"url"`
}

var youtubeID = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)
var metadataClient = &http.Client{Timeout: 6 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

func lookup(ctx context.Context, client *http.Client, provider, term string) ([]Match, error) {
	q := url.Values{}
	var endpoint string
	switch provider {
	case "youtube":
		if !youtubeID.MatchString(term) {
			return nil, errors.New("Enter an 11-character YouTube video ID.")
		}
		endpoint = "https://www.youtube.com/oembed"
		q.Set("url", "https://www.youtube.com/watch?v="+term)
		q.Set("format", "json")
	case "tvmaze":
		if strings.TrimSpace(term) == "" || len(term) > 180 {
			return nil, errors.New("Enter a series title.")
		}
		endpoint = "https://api.tvmaze.com/search/shows"
		q.Set("q", term)
	default:
		return nil, errors.New("Unsupported metadata provider.")
	}
	req, e := http.NewRequestWithContext(ctx, "GET", endpoint+"?"+q.Encode(), nil)
	if e != nil {
		return nil, e
	}
	req.Header.Set("User-Agent", "FamilyTV metadata lookup")
	res, e := client.Do(req)
	if e != nil {
		return nil, errors.New("Metadata provider unavailable.")
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("Metadata provider returned HTTP %d.", res.StatusCode)
	}
	b, e := io.ReadAll(io.LimitReader(res.Body, (2<<20)+1))
	if e != nil || len(b) > 2<<20 {
		return nil, errors.New("Invalid metadata response.")
	}
	results := []Match{}
	if provider == "youtube" {
		var obj struct {
			Title  string
			Author string `json:"author_name"`
		}
		if json.Unmarshal(b, &obj) != nil || obj.Title == "" {
			return nil, errors.New("Invalid YouTube response.")
		}
		return []Match{{obj.Title, obj.Author, term, "YouTube", "https://www.youtube.com/watch?v=" + term}}, nil
	}
	var shows []struct {
		Show struct {
			ID   int
			Name string
		}
	}
	if json.Unmarshal(b, &shows) != nil {
		return nil, errors.New("Invalid TVmaze response.")
	}
	for _, x := range shows {
		if x.Show.ID > 0 && x.Show.Name != "" {
			results = append(results, Match{x.Show.Name, "", fmt.Sprint(x.Show.ID), "TVmaze", fmt.Sprintf("https://www.tvmaze.com/shows/%d", x.Show.ID)})
		}
		if len(results) == 8 {
			break
		}
	}
	return results, nil
}
func (s *Server) metadataLookup(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	admin := s.authorized(r)
	s.mu.Unlock()
	if !admin {
		write(w, 401, map[string]string{"error": "authentication required"})
		return
	}
	matches, err := lookup(r.Context(), metadataClient, r.URL.Query().Get("provider"), r.URL.Query().Get("q"))
	if err != nil {
		write(w, 422, map[string]string{"error": err.Error()})
		return
	}
	write(w, 200, map[string]any{"matches": matches})
}
