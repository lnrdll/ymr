package source

import (
	"fmt"
	"log/slog"
	"os"
	"ymr/internal/spec"
)

// LocalLoader handles loading from the local filesystem
type LocalLoader struct {
	BaseDir  string
	SpecPath string
}

// LoadSpec reads the spec file from the local filesystem.
func (l *LocalLoader) LoadSpec(token string) (*spec.SpecConfig, error) {
	slog.Debug("Loading spec from local filesystem", "specPath", l.SpecPath)
	content, err := os.ReadFile(l.SpecPath)
	if err != nil {
		slog.Debug("Failed to read local spec", "specPath", l.SpecPath, "error", err)
		return nil, fmt.Errorf("reading local spec %s: %w", l.SpecPath, err)
	}
	return parseSpec(content)
}

// GetBasePath returns the base directory for local files.
func (l *LocalLoader) GetBasePath() string {
	return l.BaseDir
}
