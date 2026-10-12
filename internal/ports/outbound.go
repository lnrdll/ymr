package ports

import (
	config "github.com/lnrdll/ymr/internal/domain/config"
)

type SourcePort interface {
	LoadSpec(specLocation string, token string) (*config.SpecConfig, error)
	LoadSpecTemplate(specLocation string, templatePath string, token string) ([]byte, error)
	LoadTemplate(templateLocation string, token string) ([]byte, error)
	LoadParams(location string, token string) (map[string]any, error)
	LoadValidations(location string, token string) ([]config.Validation, error)
}

type ProcessorPort interface {
	ProcessContent(templateContent []byte, params map[string]any, strict bool) (string, error)
}

type ValidationEnginePort interface {
	CheckTarget(targetID string, params map[string]any) error
}

type ValidationPort interface {
	NewEngine(validations []config.Validation) (ValidationEnginePort, error)
}

type OutputPort interface {
	Write(outputs []RenderedOutput, terminalOutput bool, outputDir string) error
}

type RuntimePort interface {
	Getenv(key string) string
}
