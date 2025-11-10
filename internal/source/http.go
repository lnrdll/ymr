package source

import (
	"fmt"
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
	content, err := FetchHTTP(h.SpecURL.String(), token, true)
	if err != nil {
		return nil, fmt.Errorf("fetching spec from %s: %w", h.SpecURL.String(), err)
	}
	return parseSpec(content)
}
