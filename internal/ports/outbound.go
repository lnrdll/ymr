package ports

import (
	"github.com/lnrdll/ymr/internal/processor"
	"github.com/lnrdll/ymr/internal/source"
	"github.com/lnrdll/ymr/internal/spec"
)

type SourcePort interface {
	NewSourceLoader(path string, token string) (source.SourceLoader, error)
	NewLocalLoader(baseDir string) source.SourceLoader
	LoadTemplate(loader source.SourceLoader, templatePath string, token string) ([]byte, error)
	LoadValidations(filePath string, token string) ([]spec.Validation, error)
	LoadParams(filePath string, token string) (map[string]any, error)
}

type ProcessorPort interface {
	ProcessContent(templateContent []byte, params map[string]any) (string, error)
}

type ValidationEnginePort interface {
	CheckTarget(targetID string, params map[string]any) error
}

type ValidationPort interface {
	NewEngine(validations []spec.Validation) (ValidationEnginePort, error)
}

type OutputPort interface {
	Write(outputs []processor.RenderedOutput, terminalOutput bool, outputDir string) error
}

type RuntimePort interface {
	Getwd() (string, error)
	Getenv(key string) string
}
