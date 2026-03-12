package adapters

import (
	source "github.com/lnrdll/ymr/internal/adapters/source"
	config "github.com/lnrdll/ymr/internal/domain/config"
	"github.com/lnrdll/ymr/internal/ports"
)

type sourceAdapter struct{}

func NewSourceAdapter() ports.SourcePort {
	return sourceAdapter{}
}

func (sourceAdapter) NewSourceLoader(path string, token string) (ports.SourceLoader, error) {
	return source.NewSourceLoader(path, token)
}

func (sourceAdapter) NewLocalLoader(baseDir string) ports.SourceLoader {
	return &source.LocalLoader{BaseDir: baseDir, SpecPath: ""}
}

func (sourceAdapter) LoadTemplate(loader ports.SourceLoader, templatePath string, token string) ([]byte, error) {
	return source.LoadTemplate(loader, templatePath, token)
}

func (sourceAdapter) LoadValidations(filePath string, token string) ([]config.Validation, error) {
	return source.LoadValidations(filePath, token)
}

func (sourceAdapter) LoadParams(filePath string, token string) (map[string]any, error) {
	return source.LoadParams(filePath, token)
}
