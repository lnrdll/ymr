package source

import (
	"fmt"
	"log/slog"
	"os"

	config "github.com/lnrdll/ymr/internal/domain/config"
)

type LocalLoader struct {
	BaseDir  string
	SpecPath string
}

func (l *LocalLoader) LoadSpec(token string) (cfg *config.SpecConfig, err error) {
	slog.Debug("Loading spec from local filesystem", "specPath", l.SpecPath)

	content, err := os.ReadFile(l.SpecPath)
	if err != nil {
		return nil, fmt.Errorf("opening local spec %s: %w", l.SpecPath, err)
	}

	return parseSpec(content)
}

func (l *LocalLoader) GetBasePath() string {
	return l.BaseDir
}
