package source

import (
	"fmt"
	"log/slog"
	"path"
	"ymr/internal/spec"
)

type GithubLoader struct {
	User   string
	Repo   string
	SubDir string
	Ref    string
}

// getRawURL constructs the URL for fetching a raw file from GitHub.
func (g *GithubLoader) getRawURL(filePath string) string {
	fullPath := path.Join(g.SubDir, filePath)
	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", g.User, g.Repo, g.Ref, fullPath)
}

// LoadSpec fetches the spec file from a GitHub repository.
func (g *GithubLoader) LoadSpec(token string) (*spec.SpecConfig, error) {
	specURL := g.getRawURL("spec.yaml")
	slog.Debug("Loading spec from GitHub", "specURL", specURL)
	content, err := fetch(specURL, token, true)
	if err != nil {
		slog.Debug("Failed to fetch spec from GitHub", "specURL", specURL, "error", err)
		return nil, fmt.Errorf("fetching spec from %s: %w", specURL, err)
	}
	return parseSpec(content)
}
