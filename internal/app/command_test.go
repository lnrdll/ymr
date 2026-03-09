package app

import (
	"testing"

	processor "github.com/lnrdll/ymr/internal/adapters/processor"
	source "github.com/lnrdll/ymr/internal/adapters/source"
	config "github.com/lnrdll/ymr/internal/domain/config"
)

type fakeLoader struct {
	specConfig *config.SpecConfig
}

func (f *fakeLoader) LoadSpec(token string) (*config.SpecConfig, error) {
	if f.specConfig == nil {
		return &config.SpecConfig{}, nil
	}
	return f.specConfig, nil
}

func (f *fakeLoader) GetBasePath() string {
	return ""
}

type fakeSourcePort struct {
	newLoader      source.SourceLoader
	localLoader    source.SourceLoader
	templateBytes  []byte
	params         map[string]any
	validations    []config.Validation
	loadTplCalls   int
	newLoaderCalls int
}

func (f *fakeSourcePort) NewSourceLoader(path string, token string) (source.SourceLoader, error) {
	f.newLoaderCalls++
	if f.newLoader == nil {
		return &fakeLoader{}, nil
	}
	return f.newLoader, nil
}

func (f *fakeSourcePort) NewLocalLoader(baseDir string) source.SourceLoader {
	if f.localLoader == nil {
		return &fakeLoader{}
	}
	return f.localLoader
}

func (f *fakeSourcePort) LoadTemplate(loader source.SourceLoader, templatePath string, token string) ([]byte, error) {
	f.loadTplCalls++
	if f.templateBytes == nil {
		return []byte("x: y\n"), nil
	}
	return f.templateBytes, nil
}

func (f *fakeSourcePort) LoadValidations(filePath string, token string) ([]config.Validation, error) {
	return f.validations, nil
}

func (f *fakeSourcePort) LoadParams(filePath string, token string) (map[string]any, error) {
	if f.params == nil {
		return map[string]any{}, nil
	}
	return f.params, nil
}

type fakeProcessorPort struct {
	calls int
}

func (f *fakeProcessorPort) ProcessContent(templateContent []byte, params map[string]any) (string, error) {
	f.calls++
	return "rendered", nil
}

type fakeValidationEngine struct {
	targets []string
}

func (f *fakeValidationEngine) CheckTarget(targetID string, params map[string]any) error {
	f.targets = append(f.targets, targetID)
	return nil
}

type fakeValidationPort struct {
	engine *fakeValidationEngine
}

func (f *fakeValidationPort) NewEngine(validations []config.Validation) (ValidationEnginePort, error) {
	if f.engine == nil {
		f.engine = &fakeValidationEngine{}
	}
	return f.engine, nil
}

type fakeOutputPort struct {
	calls         int
	outputs       []processor.RenderedOutput
	terminalOut   bool
	outputDir     string
	returnedError error
}

func (f *fakeOutputPort) Write(outputs []processor.RenderedOutput, terminalOutput bool, outputDir string) error {
	f.calls++
	f.outputs = outputs
	f.terminalOut = terminalOutput
	f.outputDir = outputDir
	return f.returnedError
}

type fakeRuntimePort struct {
	cwd string
}

func (f *fakeRuntimePort) Getwd() (string, error) {
	if f.cwd == "" {
		return "/tmp", nil
	}
	return f.cwd, nil
}

func (f *fakeRuntimePort) Getenv(key string) string {
	if key == "GITHUB_TOKEN" {
		return "env-token"
	}
	return ""
}

func TestRunCommand_UsesInjectedPorts(t *testing.T) {
	t.Parallel()

	fSource := &fakeSourcePort{templateBytes: []byte("a: b\n")}
	fProcessor := &fakeProcessorPort{}
	fValidationPort := &fakeValidationPort{engine: &fakeValidationEngine{}}
	fOutput := &fakeOutputPort{}
	fRuntime := &fakeRuntimePort{cwd: "/work"}

	cmd := &runCommand{
		cfg: Config{
			OverrideTemplate: "template.yaml",
			OverrideTargets:  []string{"dev"},
			OverrideParams:   []string{"name=svc"},
		},
		deps: appDeps{
			source:     fSource,
			processor:  fProcessor,
			validation: fValidationPort,
			output:     fOutput,
			runtime:    fRuntime,
		},
	}

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if fSource.loadTplCalls != 1 {
		t.Fatalf("LoadTemplate calls = %d, want 1", fSource.loadTplCalls)
	}
	if fProcessor.calls != 1 {
		t.Fatalf("ProcessContent calls = %d, want 1", fProcessor.calls)
	}
	if fOutput.calls != 1 {
		t.Fatalf("Output write calls = %d, want 1", fOutput.calls)
	}
	if len(fOutput.outputs) != 1 {
		t.Fatalf("outputs = %d, want 1", len(fOutput.outputs))
	}
	if fOutput.outputs[0].TargetFile != "dev-template.yaml" {
		t.Fatalf("TargetFile = %q, want %q", fOutput.outputs[0].TargetFile, "dev-template.yaml")
	}
	if len(fValidationPort.engine.targets) != 1 || fValidationPort.engine.targets[0] != "dev" {
		t.Fatalf("validation targets = %#v, want [dev]", fValidationPort.engine.targets)
	}
}

func TestRunCommand_ValidateOnly_StrictUnmatchedTargetsStillFails(t *testing.T) {
	t.Parallel()

	loader := &fakeLoader{specConfig: &config.SpecConfig{
		Templates: []string{"template.yaml"},
		TargetIds: []string{"dev"},
	}}

	fSource := &fakeSourcePort{newLoader: loader}

	cmd := &runCommand{
		cfg: Config{
			IsSpecFile:      true,
			SpecFile:        "spec.yaml",
			OverrideTargets: []string{"prod"},
			Strict:          true,
			ValidateOnly:    true,
		},
		deps: appDeps{
			source:     fSource,
			processor:  &fakeProcessorPort{},
			validation: &fakeValidationPort{engine: &fakeValidationEngine{}},
			output:     &fakeOutputPort{},
			runtime:    &fakeRuntimePort{},
		},
	}

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected strict unmatched target error")
	}
	if err.Error() != "no requested targets matched spec targetIds" {
		t.Fatalf("unexpected error: %v", err)
	}
}
