package source

import (
	"fmt"
	"os"
	"regexp"
)

// IsRemotePath checks if a given path is an absolute remote URL.
func IsRemotePath(path string) bool {
	// httpRegex matches absolute http or https URLs.
	return regexp.MustCompile(`^https?://.*`).MatchString(path)
}

// LoadTemplate handles loading template content from either a remote URL or a local file.
func LoadTemplate(templatePath string, token string) ([]byte, error) {
	// 1. Check for absolute HTTP(S) URL
	if IsRemotePath(templatePath) {
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
