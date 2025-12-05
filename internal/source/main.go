package source

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"ymr/internal/spec"

	"gopkg.in/yaml.v3"
)

// SourceLoader defines the interface for any configuration source
type SourceLoader interface {
	// LoadSpec fetches and parses the spec.yaml file.
	LoadSpec(token string) (*spec.SpecConfig, error)
	// GetBasePath returns the base path for resolving relative template paths.
	GetBasePath() string
}

// NewSourceLoader determines the appropriate loader (Local, GitHub, HTTP) based on the provided path.
func NewSourceLoader(path string, token string) (SourceLoader, error) {
	slog.Debug("new source loader", "path", path)

	// GitHub
	if githubLoader, ok := ParseGitHubURL(path); ok && githubLoader.Ref != "" {
		slog.Debug("Detected GitHub source", "user", githubLoader.User, "repo", githubLoader.Repo, "subdir", githubLoader.Subdir, "ref", githubLoader.Ref)

		return &githubLoader, nil
	}

	// Local file or directory
	stat, statErr := os.Stat(path)
	if statErr == nil {
		slog.Debug("Detected local path", "path", path, "is_dir", stat.IsDir())

		if stat.IsDir() {
			// If it's a directory, default to spec.yaml
			return &LocalLoader{
				BaseDir:  path,
				SpecPath: filepath.Join(path, "spec.yaml"),
			}, nil
		} else {
			// If it's a file, use it directly
			return &LocalLoader{
				BaseDir:  filepath.Dir(path),
				SpecPath: path,
			}, nil
		}
	}

	// Direct HTTP(S) URL
	if isRemotePath(path) {
		slog.Debug("Detected HTTP source", "url", path)

		baseURL, err := url.Parse(path)
		if err != nil {
			return nil, fmt.Errorf("invalid HTTP URL: %w", err)
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
	slog.Debug("Loading template", "templatePath", templatePath)

	// If templatePath is an absolute URL, fetch it directly.
	if isRemotePath(templatePath) {
		slog.Debug("Template path is remote, fetching directly", "templatePath", templatePath)
		return fetch(templatePath, token, false)
	}

	// Determine the base path from the loader.
	basePath := loader.GetBasePath()

	slog.Debug("Using loader base path", "basePath", basePath)

	var finalPath string
	var err error

	// If templatePath is an absolute path, use it directly.
	if filepath.IsAbs(templatePath) {
		finalPath = templatePath
	} else {
		// Join the base path with the relative template path.
		// For remote paths, this correctly resolves relative URLs.
		finalPath, err = url.JoinPath(basePath, templatePath)
		if err != nil {
			slog.Debug("Error joining path", "basePath", basePath, "templatePath", templatePath, "error", err)
			return nil, fmt.Errorf("error joining path: %w", err)
		}
	}

	slog.Debug("Final template path resolved", "finalPath", finalPath)

	// Fetch the content.
	if isRemotePath(finalPath) {
		slog.Debug("Fetching remote template content", "finalPath", finalPath)
		return fetch(finalPath, token, false)
	}

	return os.ReadFile(finalPath)
}

// LoadValidations loads a YAML file containing a list of validations.
func LoadValidations(filePath string, token string) ([]spec.Validation, error) {
	data, err := fetch(filePath, token, false)
	if err != nil {
		return nil, fmt.Errorf("reading validation file '%s': %w", filePath, err)
	}

	var validations []spec.Validation
	if err := yaml.Unmarshal(data, &validations); err != nil {
		return nil, fmt.Errorf("unmarshaling validation file '%s': %w", filePath, err)
	}

	return validations, nil
}
