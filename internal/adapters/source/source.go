package source

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"

	config "github.com/lnrdll/ymr/internal/domain/config"
)

// resolver owns every rule about where a spec, template, params or
// validation file comes from. It is stateless apart from its HTTP client.
type resolver struct {
	client *http.Client
}

func newResolver(client *http.Client) resolver {
	return resolver{client: client}
}

var defaultResolver = newResolver(&http.Client{Timeout: defaultHTTPTimeout})

// LoadSpec loads the spec at specLocation: a local spec file or a directory
// holding spec.yaml, an HTTP(S) URL, or a GitHub location.
func LoadSpec(specLocation string, token string) (*config.SpecConfig, error) {
	return defaultResolver.LoadSpec(specLocation, token)
}

// LoadSpecTemplate loads a template listed in the spec at specLocation,
// resolving a relative templatePath against the spec's location.
func LoadSpecTemplate(specLocation string, templatePath string, token string) ([]byte, error) {
	return defaultResolver.LoadSpecTemplate(specLocation, templatePath, token)
}

// LoadTemplate loads a template given directly by the user, resolving a
// relative local path against the process working directory.
func LoadTemplate(templateLocation string, token string) ([]byte, error) {
	return defaultResolver.LoadTemplate(templateLocation, token)
}

func LoadParams(location string, token string) (map[string]any, error) {
	return defaultResolver.LoadParams(location, token)
}

func LoadValidations(location string, token string) ([]config.Validation, error) {
	return defaultResolver.LoadValidations(location, token)
}

// read fetches an HTTP(S) URL and reads anything else from the local
// filesystem, so a missing file fails with the OS error naming it.
func (r resolver) read(location string, token string) ([]byte, error) {
	if isRemotePath(location) {
		return r.fetch(location, token, false)
	}
	return os.ReadFile(location)
}

// acquire is read plus GitHub forms, scheme-less ones included, which are
// fetched from their raw content URL.
func (r resolver) acquire(location string, token string) ([]byte, error) {
	if _, ok := parseGitHubURL(location); ok {
		return r.fetch(location, token, false)
	}
	return r.read(location, token)
}

// specSource says where a spec is read from and what its relative
// templates resolve against.
type specSource struct {
	file    string // local path or URL of the spec itself
	baseDir string // local directory or URL relative templates join onto
	readErr string // format wrapping a read error, naming file
	local   bool   // file is read from disk rather than fetched
}

// resolveSpec checks GitHub forms first, then an existing local path, then
// HTTP(S).
func resolveSpec(specLocation string) (specSource, error) {
	if gh, ok := parseGitHubURL(specLocation); ok {
		specPath := gh.specPath()
		return specSource{
			file:    gh.rawURL(specPath),
			baseDir: gh.rawURL(path.Dir(specPath)),
			readErr: "fetching spec from GitHub %s: %w",
		}, nil
	}

	if stat, err := os.Stat(specLocation); err == nil {
		if stat.IsDir() {
			return specSource{
				file:    filepath.Join(specLocation, "spec.yaml"),
				baseDir: specLocation,
				readErr: "opening local spec %s: %w",
				local:   true,
			}, nil
		}
		return specSource{
			file:    specLocation,
			baseDir: filepath.Dir(specLocation),
			readErr: "opening local spec %s: %w",
			local:   true,
		}, nil
	}

	if isRemotePath(specLocation) {
		specURL, err := url.Parse(specLocation)
		if err != nil {
			return specSource{}, fmt.Errorf("invalid HTTP URL: %w", err)
		}
		baseURL := *specURL
		baseURL.Path = path.Dir(baseURL.Path)
		baseURL.RawQuery = ""
		baseURL.Fragment = ""
		return specSource{
			file:    specURL.String(),
			baseDir: baseURL.String(),
			readErr: "fetching spec from %s: %w",
		}, nil
	}

	return specSource{}, fmt.Errorf("could not determine source type for path: %s", specLocation)
}

func (r resolver) LoadSpec(specLocation string, token string) (*config.SpecConfig, error) {
	spec, err := resolveSpec(specLocation)
	if err != nil {
		return nil, err
	}
	slog.Debug("Loading spec", "specLocation", specLocation, "file", spec.file)

	// A local spec is never fetched, even when its path looks like a URL, and
	// a remote one is fetched as a spec, as the old loaders did.
	var content []byte
	if spec.local {
		content, err = os.ReadFile(spec.file)
	} else {
		content, err = r.fetch(spec.file, token, true)
	}
	if err != nil {
		return nil, fmt.Errorf(spec.readErr, spec.file, err)
	}

	return parseSpec(content)
}

func (r resolver) LoadSpecTemplate(specLocation string, templatePath string, token string) ([]byte, error) {
	spec, err := resolveSpec(specLocation)
	if err != nil {
		return nil, err
	}
	return r.loadTemplate(spec.baseDir, templatePath, token)
}

func (r resolver) LoadTemplate(templateLocation string, token string) ([]byte, error) {
	return r.loadTemplate("", templateLocation, token)
}

// loadTemplate joins a relative templatePath onto baseDir, a local directory
// or a URL; an empty baseDir leaves it relative to the working directory.
// Only HTTP(S) templates are fetched: a scheme-less GitHub form is read as a
// local path (fix task: github-template-refs).
func (r resolver) loadTemplate(baseDir string, templatePath string, token string) ([]byte, error) {
	slog.Debug("Loading template", "templatePath", templatePath, "baseDir", baseDir)

	// An absolute path is read from disk even when baseDir is remote (fix
	// task: remote-spec-local-template).
	if isRemotePath(templatePath) || filepath.IsAbs(templatePath) {
		return r.read(templatePath, token)
	}

	if isRemotePath(baseDir) {
		templateURL, err := url.JoinPath(baseDir, templatePath)
		if err != nil {
			return nil, fmt.Errorf("error joining path: %w", err)
		}
		return r.read(templateURL, token)
	}

	return r.read(filepath.Join(baseDir, templatePath), token)
}
