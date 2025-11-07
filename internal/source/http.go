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

func (h *HTTPLoader) getBaseURL() string {
	// Return the URL of the directory containing the spec file.
	return path.Dir(h.SpecURL.String())
}

func (h *HTTPLoader) LoadSpec(token string) (*spec.SpecConfig, error) {
	content, err := FetchHTTP(h.SpecURL.String(), token, true)
	if err != nil {
		return nil, fmt.Errorf("fetching spec from %s: %w", h.SpecURL.String(), err)
	}
	return parseSpec(content)
}
