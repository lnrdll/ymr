package source

import (
	"fmt"
	"log/slog"
	"net/url"
	"path"

	config "github.com/lnrdll/ymr/internal/domain/config"
)

type HTTPLoader struct {
	SpecURL *url.URL
}

func (h *HTTPLoader) LoadSpec(token string) (*config.SpecConfig, error) {
	slog.Debug("Loading spec from HTTP", "specURL", h.SpecURL.String())

	content, err := fetch(h.SpecURL.String(), token, true)
	if err != nil {
		return nil, fmt.Errorf("fetching spec from %s: %w", h.SpecURL.String(), err)
	}
	return parseSpec(content)
}

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
