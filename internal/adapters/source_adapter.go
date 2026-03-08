package adapters

import (
	"github.com/lnrdll/ymr/internal/ports"
	"github.com/lnrdll/ymr/internal/source"
	"github.com/lnrdll/ymr/internal/spec"
)

type sourceAdapter struct{}

func NewSourceAdapter() ports.SourcePort {
	return sourceAdapter{}
}

func (sourceAdapter) NewSourceLoader(path string, token string) (source.SourceLoader, error) {
	return source.NewSourceLoader(path, token)
}

func (sourceAdapter) NewLocalLoader(baseDir string) source.SourceLoader {
	return &source.LocalLoader{BaseDir: baseDir, SpecPath: ""}
}

func (sourceAdapter) LoadTemplate(loader source.SourceLoader, templatePath string, token string) ([]byte, error) {
	return source.LoadTemplate(loader, templatePath, token)
}

func (sourceAdapter) LoadValidations(filePath string, token string) ([]spec.Validation, error) {
	return source.LoadValidations(filePath, token)
}

func (sourceAdapter) LoadParams(filePath string, token string) (map[string]any, error) {
	return source.LoadParams(filePath, token)
}
