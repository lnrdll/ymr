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

// LoadTemplate handles loading template content from either a remote URL or a local file.
func LoadTemplate(templatePath string, token string) ([]byte, error) {
	// 1. Check for absolute HTTP(S) URL
	if isRemotePath(templatePath) {
		return FetchHTTP(templatePath, token, false)
	}

	// 2. It's not a remote path, so treat it as a local file
	// relative to the CLI's current working directory.
	if stat, err := os.Stat(templatePath); err == nil && !stat.IsDir() {
		return os.ReadFile(templatePath)
	} else if err != nil {
		cwd, _ := os.Getwd()
		return nil, fmt.Errorf("template '%s' not found locally (CWD: %s): %w", templatePath, cwd, err)
	} else {
		return nil, fmt.Errorf("template path '%s' is a directory", templatePath)
	}
}
