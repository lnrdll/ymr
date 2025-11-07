package source

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"ymr/internal/spec"
)

// SourceLoader defines the interface for any configuration source
type SourceLoader interface {
	// LoadSpec fetches and parses the spec.yaml file.
	LoadSpec(token string) (*spec.SpecConfig, error)
}

var githubSourceRegex = regexp.MustCompile(`^(?:https?://)?github\.com/([^/]+)/([^/@]+)(?:/([^@]+))?@(.+)`)

// NewSourceLoader analyzes the path and returns the correct SourceLoader.
func NewSourceLoader(path string, token string) (SourceLoader, error) {
	// 1. GitHub format: github.com/user/repo/subdir@version
	if matches := githubSourceRegex.FindStringSubmatch(path); len(matches) > 0 {
		user := matches[1]
		repo := matches[2]
		subdir := matches[3]
		version := matches[4]

		ref := version
		if version == "latest" {
			var err error
			ref, err = GetGithubDefaultBranch(user, repo, token)
			if err != nil {
				return nil, fmt.Errorf("could not resolve @latest for %s/%s: %w", user, repo, err)
			}
		}
		return &GithubLoader{
			User:   user,
			Repo:   repo,
			SubDir: subdir,
			Ref:    ref,
		}, nil
	}

	// 2. Check for local file or directory
	stat, statErr := os.Stat(path)
	if statErr == nil {
		baseDir := path
		specPath := filepath.Join(path, "spec.yaml")
		if !stat.IsDir() {
			baseDir = filepath.Dir(path)
			specPath = path
		}
		return &LocalLoader{
			BaseDir:  baseDir,
			SpecPath: specPath,
		}, nil
	}

	// 3. Check for a direct HTTP(S) URL
	if isRemotePath(path) {
		baseURL, err := url.Parse(path)
		if err != nil {
			return nil, fmt.Errorf("invalid http url: %w", err)
		}
		return &HTTPLoader{
			SpecURL: baseURL,
		}, nil
	}

	return nil, fmt.Errorf("could not determine source type for path: %s", path)
}

func LoadTemplate(loader SourceLoader, templatePath string, token string) ([]byte, error) {
	// 1. If templatePath is an absolute URL, fetch it directly.
	if isRemotePath(templatePath) {
		return FetchHTTP(templatePath, token, false)
	}

	// 2. Determine the base path from the loader type.
	var basePath string
	switch l := loader.(type) {
	case *LocalLoader:
		basePath = l.BaseDir
	case *GithubLoader:
		basePath = l.getRawURL("")
	case *HTTPLoader:
		basePath = l.getBaseURL()
	default:
		// Fallback to CWD if loader type is unknown or nil.
		cwd, _ := os.Getwd()
		basePath = cwd
	}

	// 3. Join the base path with the relative template path.
	// For remote paths, this correctly resolves relative URLs.
	finalPath, err := url.JoinPath(basePath, templatePath)
	if err != nil {
		return nil, fmt.Errorf("error joining path: %w", err)
	}

	// 4. Fetch the content.
	if isRemotePath(finalPath) {
		return FetchHTTP(finalPath, token, false)
	}
	return os.ReadFile(finalPath)
}
