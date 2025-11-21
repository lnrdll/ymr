package source

import (
	"fmt"
	"log/slog"
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
	slog.Debug("Attempting to create source loader", "path", path)
	// GitHub format: github.com/user/repo/subdir@version
	if matches := githubSourceRegex.FindStringSubmatch(path); len(matches) > 0 {
		user := matches[1]
		repo := matches[2]
		subdir := matches[3]
		ref := matches[4]

		slog.Debug("Detected GitHub source", "user", user, "repo", repo, "subdir", subdir, "ref", ref)
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
		slog.Debug("Detected local path", "path", path, "is_dir", stat.IsDir())
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
		slog.Debug("Detected HTTP source", "url", path)
		baseURL, err := url.Parse(path)
		if err != nil {
			slog.Debug("Invalid HTTP URL", "url", path, "error", err)
			return nil, fmt.Errorf("invalid http url: %w", err)
		}
		return &HTTPLoader{
			SpecURL: baseURL,
		}, nil
	}

	slog.Debug("Could not determine source type", "path", path)
	return nil, fmt.Errorf("could not determine source type for path: %s", path)
}

// LoadTemplate fetches template content from a given path, using the appropriate loader.
// It supports local files, GitHub URLs, and HTTP URLs.
func LoadTemplate(loader SourceLoader, templatePath string, token string) ([]byte, error) {
	slog.Debug("Loading template", "templatePath", templatePath)
	// If templatePath is an absolute URL, fetch it directly.
	if isRemotePath(templatePath) {
		slog.Debug("Template path is remote, fetching directly", "templatePath", templatePath)
		return fetch(templatePath, token, false)
	}

	// Determine the base path from the loader type.
	var basePath string
	switch l := loader.(type) {
	case *LocalLoader:
		basePath = l.BaseDir
		slog.Debug("Using LocalLoader base directory", "baseDir", basePath)
	case *GithubLoader:
		basePath = l.getRawURL("")
		slog.Debug("Using GithubLoader base URL", "baseURL", basePath)
	case *HTTPLoader:
		basePath = l.getBaseURL()
		slog.Debug("Using HTTPLoader base URL", "baseURL", basePath)
	default:
		// Fallback to CWD if loader type is unknown or nil.
		cwd, _ := os.Getwd()
		basePath = cwd
		slog.Debug("Unknown loader type, falling back to CWD", "cwd", basePath)
	}

	// Join the base path with the relative template path.
	// For remote paths, this correctly resolves relative URLs.
	finalPath, err := url.JoinPath(basePath, templatePath)
	if err != nil {
		slog.Debug("Error joining path", "basePath", basePath, "templatePath", templatePath, "error", err)
		return nil, fmt.Errorf("error joining path: %w", err)
	}
	slog.Debug("Final template path resolved", "finalPath", finalPath)

	// Fetch the content.
	if isRemotePath(finalPath) {
		slog.Debug("Fetching remote template content", "finalPath", finalPath)
		return fetch(finalPath, token, false)
	}
	slog.Debug("Reading local template content", "finalPath", finalPath)
	return os.ReadFile(finalPath)
}
