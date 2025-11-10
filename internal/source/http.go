package source

import (
	"fmt"
	"log/slog"
	"net/url"
	"path"
	"ymr/internal/spec"
)

// HTTPLoader handles loading from a direct http(s) URL.
type HTTPLoader struct {
	SpecURL *url.URL
}

// getBaseURL returns the base URL for resolving relative template paths.
func (h *HTTPLoader) getBaseURL() string {
	return path.Dir(h.SpecURL.String())
}

// LoadSpec fetches the spec file from a remote HTTP URL.
func (h *HTTPLoader) LoadSpec(token string) (*spec.SpecConfig, error) {
	slog.Debug("Loading spec from HTTP", "specURL", h.SpecURL.String())
	content, err := FetchHTTP(h.SpecURL.String(), token, true)
	if err != nil {
		slog.Debug("Failed to fetch spec from HTTP", "specURL", h.SpecURL.String(), "error", err)
		return nil, fmt.Errorf("fetching spec from %s: %w", h.SpecURL.String(), err)
	}
	return parseSpec(content)
}
