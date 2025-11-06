package source

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"ymr/internal/fetch"
	"ymr/internal/spec"

	"gopkg.in/yaml.v3"
)

// Regexes for matching different source types
var (
	githubSourceRegex = regexp.MustCompile(`^(?:https?://)?github\.com/([^/]+)/([^/@]+)(?:/([^@]+))?@(.+)`)
)

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
			ref, err = fetch.GetGithubDefaultBranch(user, repo, token)
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
	if IsRemotePath(path) {
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

// parseSpec is a simple helper to unmarshal
func parseSpec(content []byte) (*spec.SpecConfig, error) {
	var config spec.SpecConfig
	err := yaml.Unmarshal(content, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse spec YAML: %w", err)
	}
	return &config, nil
}
