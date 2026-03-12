package source

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"

	config "github.com/lnrdll/ymr/internal/domain/config"
	"github.com/lnrdll/ymr/internal/ports"

	"gopkg.in/yaml.v3"
)

func NewSourceLoader(path string, token string) (ports.SourceLoader, error) {
	slog.Debug("new source loader", "path", path)

	if githubLoader, ok := ParseGitHubURL(path); ok && githubLoader.Ref != "" {
		slog.Debug("Detected GitHub source", "user", githubLoader.User, "repo", githubLoader.Repo, "subdir", githubLoader.Subdir, "ref", githubLoader.Ref)
		return &githubLoader, nil
	}

	stat, statErr := os.Stat(path)
	if statErr == nil {
		slog.Debug("Detected local path", "path", path, "is_dir", stat.IsDir())

		if stat.IsDir() {
			return &LocalLoader{BaseDir: path, SpecPath: filepath.Join(path, "spec.yaml")}, nil
		}

		return &LocalLoader{BaseDir: filepath.Dir(path), SpecPath: path}, nil
	}

	if isRemotePath(path) {
		slog.Debug("Detected HTTP source", "url", path)

		baseURL, err := url.Parse(path)
		if err != nil {
			return nil, fmt.Errorf("invalid HTTP URL: %w", err)
		}
		return &HTTPLoader{SpecURL: baseURL}, nil
	}

	return nil, fmt.Errorf("could not determine source type for path: %s", path)
}

func LoadTemplate(loader ports.SourceLoader, templatePath string, token string) ([]byte, error) {
	slog.Debug("Loading template", "templatePath", templatePath)

	if isRemotePath(templatePath) {
		slog.Debug("Template path is remote, fetching directly", "templatePath", templatePath)
		return fetch(templatePath, token, false)
	}

	basePath := loader.GetBasePath()
	slog.Debug("Using loader base path", "basePath", basePath)

	var finalPath string
	var err error

	if filepath.IsAbs(templatePath) {
		finalPath = templatePath
	} else {
		if isRemotePath(basePath) {
			finalPath, err = url.JoinPath(basePath, templatePath)
			if err != nil {
				slog.Debug("Error joining path", "basePath", basePath, "templatePath", templatePath, "error", err)
				return nil, fmt.Errorf("error joining path: %w", err)
			}
		} else {
			finalPath = filepath.Join(basePath, templatePath)
		}
	}

	slog.Debug("Final template path resolved", "finalPath", finalPath)

	if isRemotePath(finalPath) {
		slog.Debug("Fetching remote template content", "finalPath", finalPath)
		return fetch(finalPath, token, false)
	}

	return os.ReadFile(finalPath)
}

func LoadValidations(filePath string, token string) ([]config.Validation, error) {
	var data []byte
	var err error
	if isRemotePath(filePath) {
		data, err = fetch(filePath, token, false)
	} else if _, ok := ParseGitHubURL(filePath); ok {
		data, err = fetch(filePath, token, false)
	} else {
		data, err = os.ReadFile(filePath)
	}
	if err != nil {
		return nil, fmt.Errorf("reading validation file '%s': %w", filePath, err)
	}

	var validations []config.Validation
	if err := yaml.Unmarshal(data, &validations); err != nil {
		return nil, fmt.Errorf("unmarshaling validation file '%s': %w", filePath, err)
	}

	return validations, nil
}
