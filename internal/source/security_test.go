package source

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestShouldUseGitHubAuth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		rawURL string
		want   bool
	}{
		{name: "github host", rawURL: "https://github.com/org/repo", want: true},
		{name: "github raw host", rawURL: "https://raw.githubusercontent.com/org/repo/main/spec.yaml", want: true},
		{name: "non github", rawURL: "https://example.com/spec.yaml", want: false},
		{name: "invalid", rawURL: "://bad-url", want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := shouldUseGitHubAuth(tt.rawURL)
			if got != tt.want {
				t.Fatalf("shouldUseGitHubAuth(%q) = %v, want %v", tt.rawURL, got, tt.want)
			}
		})
	}
}

func TestFetch_DoesNotSendGitHubHeadersToNonGitHubHosts(t *testing.T) {
	t.Parallel()

	var authHeader string
	var acceptHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		acceptHeader = r.Header.Get("Accept")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(server.Close)

	_, err := fetch(server.URL+"/spec.yaml", "secret-token", true)
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}

	if authHeader != "" {
		t.Fatalf("expected no Authorization header for non-GitHub host, got %q", authHeader)
	}
	if acceptHeader != "" {
		t.Fatalf("expected no GitHub Accept header for non-GitHub host, got %q", acceptHeader)
	}
}

func TestHTTPLoaderGetBasePath(t *testing.T) {
	t.Parallel()

	specURL, err := url.Parse("https://example.com/team/env/spec.yaml?x=1#frag")
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}

	loader := &HTTPLoader{SpecURL: specURL}
	got := loader.GetBasePath()
	want := "https://example.com/team/env"

	if got != want {
		t.Fatalf("GetBasePath() = %q, want %q", got, want)
	}
}
