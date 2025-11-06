package fetch

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
)

// Regex for GitHub blob URLs (e.g., .../user/repo/blob/branch/path/to/file)
var githubBlobRegex = regexp.MustCompile(`^https?://github\.com/([^/]+)/([^/]+)/blob/([^/]+)/(.+)`)

// Regex for GitHub repo URLs (e.g., .../user/repo or .../user/repo/tree/branch)
var githubRepoRegex = regexp.MustCompile(`^https?://github\.com/([^/]+)/([^/@]+)(?:/tree/([^/]+))?/?$`)

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

// FetchHTTP fetches a raw URL with an optional auth token.
func FetchHTTP(url string, token string, isSpec bool) ([]byte, error) {
	transformedURL := transformURL(url, isSpec)

	client := &http.Client{}
	req, err := http.NewRequest("GET", transformedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", transformedURL, err)
	}

	if token != "" {
		req.Header.Add("Authorization", "Bearer "+token)
	}
	req.Header.Add("Accept", "application/vnd.github.v3.raw")

	resp, err := client.Do(req)
	if err != nil {
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
		return nil, fmt.Errorf("bad response from server for %s: %s", transformedURL, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// GetGithubDefaultBranch fetches the default branch name from the GitHub API.
func GetGithubDefaultBranch(user, repo, token string) (string, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s", user, repo)
	client := &http.Client{}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating API request: %w", err)
	}

	if token != "" {
		req.Header.Add("Authorization", "Bearer "+token)
	}
	req.Header.Add("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("calling GitHub API: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			if err == nil {
				err = cerr
			}
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API request failed: %s", resp.Status)
	}

	var repoInfo struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&repoInfo); err != nil {
		return "", fmt.Errorf("parsing GitHub API response: %w", err)
	}

	if repoInfo.DefaultBranch == "" {
		return "", fmt.Errorf("could not determine default branch for %s/%s", user, repo)
	}
	return repoInfo.DefaultBranch, nil
}
