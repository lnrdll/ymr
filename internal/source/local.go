package source

import (
	"fmt"
	"os"
	"ymr/internal/spec"
)

// LocalLoader handles loading from the local filesystem
type LocalLoader struct {
	BaseDir  string
	SpecPath string
}

func (l *LocalLoader) LoadSpec(token string) (*spec.SpecConfig, error) {
	content, err := os.ReadFile(l.SpecPath)
	if err != nil {
		return nil, fmt.Errorf("reading local spec %s: %w", l.SpecPath, err)
	}
	return parseSpec(content)
}
