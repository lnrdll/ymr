package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidate_PassesWhenValidationsPass(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	templatePath := filepath.Join(workDir, "t.yaml")
	specPath := filepath.Join(workDir, "spec.yaml")

	err := os.WriteFile(templatePath, []byte(strings.TrimSpace(`
key: default # from-param: {{ .name }}
`)+"\n"), 0644)
	if err != nil {
		t.Fatalf("write template: %v", err)
	}

	err = os.WriteFile(specPath, []byte(strings.TrimSpace(`
templates:
  - t.yaml
targetIds:
  - dev
parameters:
  - targetId: ["dev"]
    values:
      minScale: 1
validations:
  - rule: "params.minScale >= 1"
    message: "minScale must be >= 1"
    targetId: ["dev"]
`)+"\n"), 0644)
	if err != nil {
		t.Fatalf("write spec: %v", err)
	}

	cfg := Config{IsSpecFile: true, SpecFile: specPath}
	if err := Validate(cfg); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestValidate_FailsWhenValidationFails(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	templatePath := filepath.Join(workDir, "t.yaml")
	specPath := filepath.Join(workDir, "spec.yaml")

	err := os.WriteFile(templatePath, []byte("key: v\n"), 0644)
	if err != nil {
		t.Fatalf("write template: %v", err)
	}

	err = os.WriteFile(specPath, []byte(strings.TrimSpace(`
templates:
  - t.yaml
targetIds:
  - dev
parameters:
  - targetId: ["dev"]
    values:
      minScale: 0
validations:
  - rule: "params.minScale >= 1"
    message: "minScale must be >= 1"
    targetId: ["dev"]
`)+"\n"), 0644)
	if err != nil {
		t.Fatalf("write spec: %v", err)
	}

	cfg := Config{IsSpecFile: true, SpecFile: specPath}
	if err := Validate(cfg); err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestValidate_Strict_FailsOnMissingTemplate(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	specPath := filepath.Join(workDir, "spec.yaml")

	err := os.WriteFile(specPath, []byte(strings.TrimSpace(`
templates:
  - missing.yaml
targetIds:
  - dev
parameters: []
validations: []
`)+"\n"), 0644)
	if err != nil {
		t.Fatalf("write spec: %v", err)
	}

	cfg := Config{IsSpecFile: true, SpecFile: specPath, Strict: true}
	if err := Validate(cfg); err == nil {
		t.Fatalf("expected strict error")
	}
}
