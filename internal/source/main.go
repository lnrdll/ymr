package source

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"ymr/internal/spec"
)

// SourceLoader defines the interface for any configuration source
type SourceLoader interface {
	// LoadSpec fetches and parses the spec.yaml file.
	LoadSpec(token string) (*spec.SpecConfig, error)
}

// NewSourceLoader determines the appropriate loader (Local, GitHub, HTTP) based on the provided path.
func NewSourceLoader(path string, token string) (SourceLoader, error) {
	// GitHub format: github.com/user/repo/subdir@version
	if matches := githubSourceRegex.FindStringSubmatch(path); len(matches) > 0 {
		user := matches[1]
		repo := matches[2]
		subdir := matches[3]
		ref := matches[4]

		return &GithubLoader{
			User:   user,
			Repo:   repo,
			SubDir: subdir,
			Ref:    ref,
		}, nil
	}

	// Check for local file or directory
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

	// Check for a direct HTTP(S) URL
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

// LoadTemplate fetches template content from a given path, using the appropriate loader.
// It supports local files, GitHub URLs, and HTTP URLs.
func LoadTemplate(loader SourceLoader, templatePath string, token string) ([]byte, error) {
	// If templatePath is an absolute URL, fetch it directly.
	if isRemotePath(templatePath) {
		return FetchHTTP(templatePath, token, false)
	}

	// Determine the base path from the loader type.
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

	// Join the base path with the relative template path.
	// For remote paths, this correctly resolves relative URLs.
	finalPath, err := url.JoinPath(basePath, templatePath)
	if err != nil {
		return nil, fmt.Errorf("error joining path: %w", err)
	}

	// Fetch the content.
	if isRemotePath(finalPath) {
		return FetchHTTP(finalPath, token, false)
	}
	return os.ReadFile(finalPath)
}
