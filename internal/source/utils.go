package source

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"ymr/internal/spec"

	"gopkg.in/yaml.v3"
)

// Regex for GitHub blob URLs (e.g., .../user/repo/blob/branch/path/to/file)
var githubBlobRegex = regexp.MustCompile(`^https?://github\.com/([^/]+)/([^/]+)/blob/([^/]+)/(.+)`)

// Regex for GitHub repo URLs (e.g., .../user/repo or .../user/repo/tree/branch)
var githubRepoRegex = regexp.MustCompile(`^https?://github\.com/([^/]+)/([^/@]+)(?:/tree/([^/]+))?/?$`)

// Regex for GitHub format: github.com/user/repo/subdir@version
var githubSourceRegex = regexp.MustCompile(`^(?:https?://)?github\.com/([^/]+)/([^/@]+)(?:/([^@]+))?@(.+)`)

// isRemotePath checks if a given path is an absolute remote URL.
func isRemotePath(path string) bool {
	// httpRegex matches absolute http or https URLs.
	return regexp.MustCompile(`^https?://.*`).MatchString(path)
}

// parseSpec unmarshals the spec file content into a SpecConfig struct.
func parseSpec(content []byte) (*spec.SpecConfig, error) {
	slog.Debug("Parsing spec content")
	var config spec.SpecConfig
	err := yaml.Unmarshal(content, &config)
	if err != nil {
		slog.Debug("Failed to parse spec YAML", "error", err)
		return nil, fmt.Errorf("failed to parse spec YAML: %w", err)
	}
	return &config, nil
}

// transformURL converts GitHub blob and repo URLs to raw content URLs.
func transformURL(path string, isSpec bool) string {
	slog.Debug("Transforming URL", "originalURL", path, "isSpec", isSpec)
	// Check for `github.com/user/repo/blob/branch/path`
	if matches := githubBlobRegex.FindStringSubmatch(path); len(matches) == 5 {
		// Transform to: raw.githubusercontent.com/user/repo/branch/path
		transformedURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", matches[1], matches[2], matches[3], matches[4])
		slog.Debug("Transformed GitHub blob URL", "original", path, "transformed", transformedURL)
		return transformedURL
	}

	// Check for `github.com/user/repo` or `.../tree/branch` (ONLY if loading a spec)
	if isSpec {
		if matches := githubRepoRegex.FindStringSubmatch(path); len(matches) > 0 {
			user := matches[1]
			repo := matches[2]
			branch := "main" // Default branch
			if len(matches) == 4 && matches[3] != "" {
				branch = matches[3]
			}
			// Transform to: raw.githubusercontent.com/user/repo/branch/spec.yaml
			transformedURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/spec.yaml", user, repo, branch)
			slog.Debug("Transformed GitHub repo URL for spec", "original", path, "transformed", transformedURL)
			return transformedURL
		}
	}
	slog.Debug("No URL transformation applied", "url", path)
	return path
}

// fetch fetches content from a URL, with optional GitHub token authentication.
func fetch(url string, token string, isSpec bool) ([]byte, error) {
	slog.Debug("Fetching HTTP content", "url", url, "isSpec", isSpec)
	transformedURL := transformURL(url, isSpec)

	client := &http.Client{}
	req, err := http.NewRequest("GET", transformedURL, nil)
	if err != nil {
		slog.Debug("Failed to create HTTP request", "url", transformedURL, "error", err)
		return nil, fmt.Errorf("failed to create request for %s: %w", transformedURL, err)
	}

	if token != "" {
		req.Header.Add("Authorization", "Bearer "+token)
	}
	req.Header.Add("Accept", "application/vnd.github.v3.raw")
	req.Header.Add("Cache-control", "no-cache")

	resp, err := client.Do(req)
	if err != nil {
		slog.Debug("Failed to fetch remote content", "url", transformedURL, "error", err)
		return nil, fmt.Errorf("failed to fetch remote content from %s: %w", transformedURL, err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			if err == nil {
				err = cerr
			}
		}
	}()

	if resp.StatusCode != http.StatusOK {
		slog.Debug("Bad response from server", "url", transformedURL, "status", resp.Status)
		return nil, fmt.Errorf("bad response from server for %s: %s", transformedURL, resp.Status)
	}

	return io.ReadAll(resp.Body)
}
