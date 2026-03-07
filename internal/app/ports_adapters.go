package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lnrdll/ymr/internal/processor"
	"github.com/lnrdll/ymr/internal/source"
	"github.com/lnrdll/ymr/internal/spec"
	"github.com/lnrdll/ymr/internal/validation"
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

type sourceAdapter struct{}

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

type processorAdapter struct{}

func (processorAdapter) ProcessContent(templateContent []byte, params map[string]any) (string, error) {
	return processor.ProcessContent(templateContent, params)
}

type validationAdapter struct{}

func (validationAdapter) NewEngine(validations []spec.Validation) (ValidationEnginePort, error) {
	return validation.NewEngine(validations)
}

type outputAdapter struct{}

func (outputAdapter) Write(outputs []processor.RenderedOutput, terminalOutput bool, outputDir string) error {
	if terminalOutput {
		for _, output := range outputs {
			fmt.Print(output.Content)
		}
		return nil
	}

	if outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("creating output directory '%s': %w", outputDir, err)
		}
	}

	var writeErrs []error
	for _, output := range outputs {
		outPath := filepath.Join(outputDir, output.TargetFile)
		err := os.WriteFile(outPath, []byte(output.Content), 0644)
		if err != nil {
			writeErrs = append(writeErrs, fmt.Errorf("writing output file '%s': %w", outPath, err))
		}
	}

	if len(writeErrs) > 0 {
		return errors.Join(writeErrs...)
	}

	return nil
}

type runtimeAdapter struct{}

func (runtimeAdapter) Getwd() (string, error) {
	return os.Getwd()
}

func (runtimeAdapter) Getenv(key string) string {
	return os.Getenv(key)
}

type appDeps struct {
	source     SourcePort
	processor  ProcessorPort
	validation ValidationPort
	output     OutputPort
	runtime    RuntimePort
}

func newDefaultDeps() appDeps {
	return appDeps{
		source:     sourceAdapter{},
		processor:  processorAdapter{},
		validation: validationAdapter{},
		output:     outputAdapter{},
		runtime:    runtimeAdapter{},
	}
}
