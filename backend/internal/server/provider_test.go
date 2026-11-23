package server

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestMetadataProviders(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		var body string
		switch r.URL.Host {
		case "api.tvmaze.com":
			if r.URL.Query().Get("q") != "Example & Friends" {
				t.Fatal(r.URL)
			}
			body = `[{"show":{"id":123,"name":"Example"}}]`
		case "www.youtube.com":
			if r.URL.Query().Get("url") != "https://www.youtube.com/watch?v=example1234" {
				t.Fatal(r.URL)
			}
			body = `{"title":"Example","author_name":"Creator"}`
		default:
			t.Fatal(r.URL)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	ctx := context.Background()
	a, e := lookup(ctx, client, "tvmaze", "Example & Friends")
	if e != nil || len(a) != 1 || a[0].URL != "https://www.tvmaze.com/shows/123" {
		t.Fatal(a, e)
	}
	a, e = lookup(ctx, client, "youtube", "example1234")
	if e != nil || a[0].Creator != "Creator" {
		t.Fatal(a, e)
	}
	if _, e = lookup(ctx, client, "youtube", "invalid input"); e == nil || calls != 2 {
		t.Fatal("invalid ID should not make a request")
	}
}
