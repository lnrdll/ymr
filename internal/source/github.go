package source

import (
	"fmt"
	"path"
	"ymr/internal/fetch"
	"ymr/internal/spec"
)

type GithubLoader struct {
	User   string
	Repo   string
	SubDir string
	Ref    string
}

func (g *GithubLoader) getRawURL(filePath string) string {
	fullPath := path.Join(g.SubDir, filePath)
	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", g.User, g.Repo, g.Ref, fullPath)
}

func (g *GithubLoader) LoadSpec(token string) (*spec.SpecConfig, error) {
	specURL := g.getRawURL("spec.yaml")
	content, err := fetch.FetchHTTP(specURL, token, true)
	if err != nil {
		return nil, fmt.Errorf("fetching spec from %s: %w", specURL, err)
	}
	return parseSpec(content)
}

func (g *GithubLoader) LoadTemplate(templatePath string, token string) ([]byte, error) {
	return loadTemplateContent(templatePath, token)
}
