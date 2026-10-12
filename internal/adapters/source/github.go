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

func (g *githubLocation) rawContentURL(isSpec bool) string {
	if g.Path != "" {
		return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", g.User, g.Repo, g.Ref, g.Path)
	}

	if isSpec && g.Repo != "" {
		branch := g.Ref
		if branch == "" {
			branch = "main"
		}
		return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/spec.yaml", g.User, g.Repo, branch)
	}

	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", g.User, g.Repo, g.Ref, path.Join(g.Subdir))
}

// specPath is the repo path of the spec a GitHub location names: a blob's
// file, an @ref path ending in .yaml or .yml, or else spec.yaml inside the
// named directory.
func (g *githubLocation) specPath() string {
	if g.Path != "" {
		return g.Path
	}
	if ext := path.Ext(g.Subdir); ext == ".yaml" || ext == ".yml" {
		return g.Subdir
	}
	return path.Join(g.Subdir, "spec.yaml")
}

func (g *githubLocation) rawURL(repoPath string) string {
	return "https://raw.githubusercontent.com/" + path.Join(g.User, g.Repo, g.Ref, repoPath)
}
