package source

import (
	"fmt"
	"log/slog"
	"net/url"
	"path"

	"github.com/lnrdll/ymr/internal/spec"
)

// HTTPLoader handles loading from a direct http(s) URL.
type HTTPLoader struct {
	SpecURL *url.URL
}

// LoadSpec fetches the spec file from a remote HTTP URL.
func (h *HTTPLoader) LoadSpec(token string) (*spec.SpecConfig, error) {
	slog.Debug("Loading spec from HTTP", "specURL", h.SpecURL.String())

	content, err := fetch(h.SpecURL.String(), token, true)
	if err != nil {
		return nil, fmt.Errorf("fetching spec from %s: %w", h.SpecURL.String(), err)
	}
	return parseSpec(content)
}

// GetBasePath returns the base URL for resolving relative template paths.
func (h *HTTPLoader) GetBasePath() string {
	if h.SpecURL == nil {
		return ""
	}

	baseURL := *h.SpecURL
	baseURL.Path = path.Dir(baseURL.Path)
	baseURL.RawQuery = ""
	baseURL.Fragment = ""

	return baseURL.String()
}
