package app

import (
	"reflect"
	"testing"

	config "github.com/lnrdll/ymr/internal/domain/config"
	"github.com/lnrdll/ymr/internal/ports"
)

type specTemplateCall struct {
	specLocation string
	templatePath string
}

type fakeSourcePort struct {
	specConfig        *config.SpecConfig
	templateBytes     []byte
	params            map[string]any
	validations       []config.Validation
	templateCalls     []string
	specTemplateCalls []specTemplateCall
}

func (f *fakeSourcePort) LoadSpec(specLocation string, token string) (*config.SpecConfig, error) {
	if f.specConfig == nil {
		return &config.SpecConfig{}, nil
	}
	return f.specConfig, nil
}

func (f *fakeSourcePort) LoadSpecTemplate(specLocation string, templatePath string, token string) ([]byte, error) {
	f.specTemplateCalls = append(f.specTemplateCalls, specTemplateCall{specLocation: specLocation, templatePath: templatePath})
	return f.template(), nil
}

func (f *fakeSourcePort) LoadTemplate(templateLocation string, token string) ([]byte, error) {
	f.templateCalls = append(f.templateCalls, templateLocation)
	return f.template(), nil
}

func (f *fakeSourcePort) template() []byte {
	if f.templateBytes == nil {
		return []byte("x: y\n")
	}
	return f.templateBytes
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

func (f *fakeProcessorPort) ProcessContent(templateContent []byte, params map[string]any, strict bool) (string, error) {
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
	outputs       []ports.RenderedOutput
	terminalOut   bool
	outputDir     string
	returnedError error
}

func (f *fakeOutputPort) Write(outputs []ports.RenderedOutput, terminalOutput bool, outputDir string) error {
	f.calls++
	f.outputs = outputs
	f.terminalOut = terminalOutput
	f.outputDir = outputDir
	return f.returnedError
}

type fakeRuntimePort struct{}

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
	fRuntime := &fakeRuntimePort{}

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

	if len(fSource.templateCalls) != 1 || fSource.templateCalls[0] != "template.yaml" {
		t.Fatalf("LoadTemplate calls = %#v, want [template.yaml]", fSource.templateCalls)
	}
	if len(fSource.specTemplateCalls) != 0 {
		t.Fatalf("LoadSpecTemplate calls = %#v, want none", fSource.specTemplateCalls)
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

	fSource := &fakeSourcePort{specConfig: &config.SpecConfig{
		Templates: []string{"template.yaml"},
		TargetIds: []string{"dev"},
	}}

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

func TestRunCommand_SpecModeTemplateOrigin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		overrideTemplate  string
		wantTemplateCalls []string
		wantSpecTemplates []specTemplateCall
	}{
		{
			name:              "spec templates resolve against the spec",
			wantSpecTemplates: []specTemplateCall{{specLocation: "specs/app", templatePath: "deploy.yaml"}},
		},
		{
			name:              "template override resolves against the working directory",
			overrideTemplate:  "override.yaml",
			wantTemplateCalls: []string{"override.yaml"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fSource := &fakeSourcePort{specConfig: &config.SpecConfig{
				Templates: []string{"deploy.yaml"},
				TargetIds: []string{"dev"},
			}}
			fOutput := &fakeOutputPort{}

			cmd := &runCommand{
				cfg: Config{
					IsSpecFile:       true,
					SpecFile:         "specs/app",
					OverrideTemplate: tt.overrideTemplate,
				},
				deps: appDeps{
					source:     fSource,
					processor:  &fakeProcessorPort{},
					validation: &fakeValidationPort{engine: &fakeValidationEngine{}},
					output:     fOutput,
					runtime:    &fakeRuntimePort{},
				},
			}

			if err := cmd.Execute(); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			if !reflect.DeepEqual(fSource.templateCalls, tt.wantTemplateCalls) {
				t.Fatalf("LoadTemplate calls = %#v, want %#v", fSource.templateCalls, tt.wantTemplateCalls)
			}
			if !reflect.DeepEqual(fSource.specTemplateCalls, tt.wantSpecTemplates) {
				t.Fatalf("LoadSpecTemplate calls = %#v, want %#v", fSource.specTemplateCalls, tt.wantSpecTemplates)
			}
			if len(fOutput.outputs) != 1 {
				t.Fatalf("outputs = %d, want 1", len(fOutput.outputs))
			}
		})
	}
}
