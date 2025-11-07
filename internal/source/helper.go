package source

import (
	"fmt"
	"regexp"
	"ymr/internal/spec"

	"gopkg.in/yaml.v3"
)

// Regex for GitHub blob URLs (e.g., .../user/repo/blob/branch/path/to/file)
var githubBlobRegex = regexp.MustCompile(`^https?://github\.com/([^/]+)/([^/]+)/blob/([^/]+)/(.+)`)

// Regex for GitHub repo URLs (e.g., .../user/repo or .../user/repo/tree/branch)
var githubRepoRegex = regexp.MustCompile(`^https?://github\.com/([^/]+)/([^/@]+)(?:/tree/([^/]+))?/?$`)

// isRemotePath checks if a given path is an absolute remote URL.
func isRemotePath(path string) bool {
	// httpRegex matches absolute http or https URLs.
	return regexp.MustCompile(`^https?://.*`).MatchString(path)
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

// transformURL is a new internal helper
func transformURL(path string, isSpec bool) string {
	// 1. Check for `github.com/user/repo/blob/branch/path`
	if matches := githubBlobRegex.FindStringSubmatch(path); len(matches) == 5 {
		// Transform to: raw.githubusercontent.com/user/repo/branch/path
		return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", matches[1], matches[2], matches[3], matches[4])
	}

	// 2. Check for `github.com/user/repo` or `.../tree/branch` (ONLY if loading a spec)
	if isSpec {
		if matches := githubRepoRegex.FindStringSubmatch(path); len(matches) > 0 {
			user := matches[1]
			repo := matches[2]
			branch := "main" // Default branch
			if len(matches) == 4 && matches[3] != "" {
				branch = matches[3]
			}
			// Transform to: raw.githubusercontent.com/user/repo/branch/spec.yaml
			return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/spec.yaml", user, repo, branch)
		}
	}
	return path
}
