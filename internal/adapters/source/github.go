package source

import (
	"fmt"
	"path"
)

type githubLocation struct {
	Path   string
	Ref    string
	Repo   string
	Subdir string
	User   string
}

func (g *githubLocation) rawContentURL(relativePath string, isSpec bool) string {
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
