package source

import (
	"fmt"
	"log/slog"
	"path"
	"ymr/internal/spec"
)

type GithubLoader struct {
	Path   string
	Ref    string // Consolidated Git reference (branch, tag, commit)
	Repo   string
	Subdir string
	User   string
}

// LoadSpec fetches the spec file from a GitHub repository.
func (g *GithubLoader) LoadSpec(token string) (*spec.SpecConfig, error) {
	specURL := g.GetRawContentURL("spec.yaml", true)
	slog.Debug("Loading spec from GitHub", "specURL", specURL)
	content, err := fetch(specURL, token, true)
	if err != nil {
		slog.Debug("Failed to fetch spec from GitHub", "specURL", specURL, "error", err)
		return nil, fmt.Errorf("fetching spec from %s: %w", specURL, err)
	}
	return parseSpec(content)
}

// GetRawContentURL constructs the URL for fetching a raw file from GitHub.
// It handles different scenarios: specific relative paths, blob URLs, and repo-root spec files.
func (g *GithubLoader) GetRawContentURL(relativePath string, isSpec bool) string {
	// If it's a blob URL (g.Path is set), prioritize that.
	if g.Path != "" && relativePath == "" {
		return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", g.User, g.Repo, g.Ref, g.Path)
	}

	// If it's a repo URL for a spec and no specific relativePath is given, default to spec.yaml in root.
	if isSpec && g.Repo != "" && relativePath == "" {
		branch := g.Ref
		if branch == "" {
			branch = "main"
		}
		return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/spec.yaml", g.User, g.Repo, branch)
	}

	// Otherwise, construct the URL for a specific relative path within the repository.
	fullPath := path.Join(g.Subdir, relativePath)
	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", g.User, g.Repo, g.Ref, fullPath)
}

// GetBasePath returns the base URL for resolving relative template paths in a GitHub repository.
func (g *GithubLoader) GetBasePath() string {
	return g.GetRawContentURL("", false) // Base path for templates, not necessarily a spec
}
