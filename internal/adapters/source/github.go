package source

import (
	"fmt"
	"log/slog"
	"path"

	config "github.com/lnrdll/ymr/internal/domain/config"
)

type GithubLoader struct {
	Path   string
	Ref    string
	Repo   string
	Subdir string
	User   string
}

func (g *GithubLoader) LoadSpec(token string) (*config.SpecConfig, error) {
	specURL := g.GetRawContentURL("spec.yaml", true)
	slog.Debug("Loading spec from GitHub", "specURL", specURL)

	content, err := fetch(specURL, token, true)
	if err != nil {
		return nil, fmt.Errorf("fetching spec from GitHub %s: %w", specURL, err)
	}

	return parseSpec(content)
}

func (g *GithubLoader) GetBasePath() string {
	return g.GetRawContentURL("", false)
}

func (g *GithubLoader) GetRawContentURL(relativePath string, isSpec bool) string {
	if g.Path != "" && relativePath == "" {
		return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", g.User, g.Repo, g.Ref, g.Path)
	}

	if isSpec && g.Repo != "" && relativePath == "" {
		branch := g.Ref
		if branch == "" {
			branch = "main"
		}
		return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/spec.yaml", g.User, g.Repo, branch)
	}

	fullPath := path.Join(g.Subdir, relativePath)
	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", g.User, g.Repo, g.Ref, fullPath)
}
