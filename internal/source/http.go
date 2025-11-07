package source

import (
	"fmt"
	"net/url"
	"ymr/internal/spec"
)

// HTTPLoader handles loading from a direct http(s) URL.
type HTTPLoader struct {
	SpecURL *url.URL
}

func (h *HTTPLoader) LoadSpec(token string) (*spec.SpecConfig, error) {
	content, err := FetchHTTP(h.SpecURL.String(), token, true)
	if err != nil {
		return nil, fmt.Errorf("fetching spec from %s: %w", h.SpecURL.String(), err)
	}
	return parseSpec(content)
}
