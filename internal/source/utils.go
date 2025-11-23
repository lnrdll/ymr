package source

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"ymr/internal/spec"

	"gopkg.in/yaml.v3"
)

const bufferSize = 8192

// Regex for GitHub blob URLs (e.g., .../user/repo/blob/branch/path/to/file)
var githubBlobRegex = regexp.MustCompile(`^https?://github\.com/([^/]+)/([^/]+)/blob/([^/]+)/(.+)`)

// Regex for GitHub repo URLs (e.g., .../user/repo or .../user/repo/tree/branch)
var githubRepoRegex = regexp.MustCompile(`^https?://github\.com/([^/]+)/([^/@]+)(?:/tree/([^/]+))?/?$`)

// Regex for GitHub format: github.com/user/repo/subdir@version
var githubSourceRegex = regexp.MustCompile(`^(?:https?://)?github\.com/([^/]+)/([^/@]+)(?:/([^@]+))?@(.+)`)

// ParseGitHubURL parses a GitHub URL string into a GithubLoader struct.
func ParseGitHubURL(url string) (GithubLoader, bool) {
	if matches := githubBlobRegex.FindStringSubmatch(url); len(matches) == 5 {
		return GithubLoader{
			User: matches[1],
			Repo: matches[2],
			Ref:  matches[3], // Use branch as ref for blob URLs
			Path: matches[4],
		}, true
	}

	if matches := githubRepoRegex.FindStringSubmatch(url); len(matches) > 0 {
		branch := "main"
		if len(matches) == 4 && matches[3] != "" {
			branch = matches[3]
		}
		return GithubLoader{
			User: matches[1],
			Repo: matches[2],
			Ref:  branch, // Use branch as ref for repo URLs
		}, true
	}

	if matches := githubSourceRegex.FindStringSubmatch(url); len(matches) == 5 {
		return GithubLoader{
			User:   matches[1],
			Repo:   matches[2],
			Subdir: matches[3],
			Ref:    matches[4], // Use version as ref for source URLs
		}, true
	}

	return GithubLoader{}, false
}

// isRemotePath checks if a given path is an absolute remote URL.
func isRemotePath(path string) bool {
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

// fetch fetches content from a URL, with optional GitHub token authentication.
func fetch(url string, token string, isSpec bool) ([]byte, error) {
	slog.Debug("Fetching HTTP content", "url", url, "isSpec", isSpec)

	var transformedURL string
	if ghl, ok := ParseGitHubURL(url); ok {
		transformedURL = ghl.GetRawContentURL("", isSpec)
	} else {
		transformedURL = url
	}

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

	reader := bufio.NewReaderSize(resp.Body, bufferSize)
	return io.ReadAll(reader)
}
