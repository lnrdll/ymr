package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPreparePlan_SpecLess(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	templatePath := filepath.Join(workDir, "t.yaml")
	err := os.WriteFile(templatePath, []byte("key: v\n"), 0644)
	if err != nil {
		t.Fatalf("write template: %v", err)
	}

	cfg := Config{
		IsSpecFile:       false,
		OverrideTemplate: templatePath,
		OverrideTargets:  []string{"dev"},
		OverrideParams:   []string{"name=myapp"},
	}

	plan, err := preparePlan(cfg)
	if err != nil {
		t.Fatalf("preparePlan: %v", err)
	}

	if len(plan.SpecConfig.Templates) != 1 || plan.SpecConfig.Templates[0] != templatePath {
		t.Fatalf("unexpected templates: %#v", plan.SpecConfig.Templates)
	}
	if len(plan.Targets) != 1 || plan.Targets[0] != "dev" {
		t.Fatalf("unexpected targets: %#v", plan.Targets)
	}
	if got, ok := plan.ParamsOverride["name"]; !ok || got != "myapp" {
		t.Fatalf("expected params override name=myapp, got %#v", plan.ParamsOverride)
	}
}

func TestPreparePlan_LoadsValidationOverride(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	templatePath := filepath.Join(workDir, "t.yaml")
	specPath := filepath.Join(workDir, "spec.yaml")
	validationPath := filepath.Join(workDir, "validations.yaml")

	err := os.WriteFile(templatePath, []byte("key: v\n"), 0644)
	if err != nil {
		t.Fatalf("write template: %v", err)
	}

	err = os.WriteFile(specPath, []byte("templates:\n  - t.yaml\ntargetIds:\n  - dev\nparameters: []\nvalidations:\n  - rule: \"params.a == 1\"\n    message: \"spec\"\n    targetId: [\"dev\"]\n"), 0644)
	if err != nil {
		t.Fatalf("write spec: %v", err)
	}

	err = os.WriteFile(validationPath, []byte("- rule: \"params.a == 2\"\n  message: \"override\"\n  targetId: [\"dev\"]\n"), 0644)
	if err != nil {
		t.Fatalf("write validations: %v", err)
	}

	cfg := Config{
		IsSpecFile:     true,
		SpecFile:       specPath,
		ValidationFile: validationPath,
	}

	plan, err := preparePlan(cfg)
	if err != nil {
		t.Fatalf("preparePlan: %v", err)
	}

	if plan.Loader == nil {
		t.Fatalf("expected loader to be set")
	}
	if len(plan.Validations) != 1 || plan.Validations[0].Message != "override" {
		t.Fatalf("expected validation override to be loaded, got %#v", plan.Validations)
	}
}
