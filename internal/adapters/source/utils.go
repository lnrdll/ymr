package source

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	config "github.com/lnrdll/ymr/internal/domain/config"

	"gopkg.in/yaml.v3"
)

const defaultHTTPTimeout = 30 * time.Second

var githubBlobRegex = regexp.MustCompile(`^https?://github\.com/([^/]+)/([^/]+)/blob/([^/]+)/(.+)`)
var githubRepoRegex = regexp.MustCompile(`^https?://github\.com/([^/]+)/([^/@]+)(?:/tree/([^/]+))?/?$`)
var githubSourceRegex = regexp.MustCompile(`^(?:https?://)?github\.com/([^/]+)/([^/@]+)(?:/([^@]+))?@(.+)`)

func ParseGitHubURL(rawURL string) (GithubLoader, bool) {
	if matches := githubBlobRegex.FindStringSubmatch(rawURL); len(matches) == 5 {
		return GithubLoader{User: matches[1], Repo: matches[2], Ref: matches[3], Path: matches[4]}, true
	}

	if matches := githubRepoRegex.FindStringSubmatch(rawURL); len(matches) > 0 {
		branch := "main"
		if len(matches) == 4 && matches[3] != "" {
			branch = matches[3]
		}
		return GithubLoader{User: matches[1], Repo: matches[2], Ref: branch}, true
	}

	if matches := githubSourceRegex.FindStringSubmatch(rawURL); len(matches) == 5 {
		return GithubLoader{User: matches[1], Repo: matches[2], Subdir: matches[3], Ref: matches[4]}, true
	}

	return GithubLoader{}, false
}

func isRemotePath(path string) bool {
	return strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
}

func parseSpec(content []byte) (*config.SpecConfig, error) {
	slog.Debug("Parsing spec content")

	var cfg config.SpecConfig
	err := yaml.Unmarshal(content, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse spec YAML: %w", err)
	}

	return &cfg, nil
}

func fetch(rawURL string, token string, isSpec bool) ([]byte, error) {
	slog.Debug("Fetching HTTP content", "url", rawURL, "isSpec", isSpec)

	transformedURL := rawURL
	if ghl, ok := ParseGitHubURL(rawURL); ok {
		transformedURL = ghl.GetRawContentURL("", isSpec)
	}

	client := &http.Client{Timeout: defaultHTTPTimeout}
	req, err := http.NewRequest("GET", transformedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request for %s: %w", transformedURL, err)
	}

	if token != "" && shouldUseGitHubAuth(transformedURL) {
		req.Header.Add("Authorization", "Bearer "+token)
	}
	if shouldUseGitHubAuth(transformedURL) {
		req.Header.Add("Accept", "application/vnd.github.v3.raw")
	}
	req.Header.Add("Cache-control", "no-cache")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch remote content from %s: %w", transformedURL, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad response from server for %s: %s", transformedURL, resp.Status)
	}

	return io.ReadAll(resp.Body)
}

func shouldUseGitHubAuth(rawURL string) bool {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	host := strings.ToLower(parsedURL.Hostname())
	return host == "github.com" || host == "raw.githubusercontent.com"
}
