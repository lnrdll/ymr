package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlan_SpecMode_PrintsOutputs(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	specPath := filepath.Join(workDir, "spec.yaml")

	err := os.WriteFile(specPath, []byte(strings.TrimSpace(`
templates:
  - base/a.yaml
  - b.yaml
targetIds:
  - dev
  - prd
parameters: []
validations: []
`)+"\n"), 0644)
	if err != nil {
		t.Fatalf("write spec: %v", err)
	}

	plan, err := Plan(Config{IsSpecFile: true, SpecFile: specPath, OutputDir: "rendered"})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	if !strings.Contains(plan, "Mode: spec") {
		t.Fatalf("expected mode line, got:\n%s", plan)
	}
	if !strings.Contains(plan, "Destination: rendered") {
		t.Fatalf("expected destination line, got:\n%s", plan)
	}
	if !strings.Contains(plan, "Outputs (4):") {
		t.Fatalf("expected outputs count, got:\n%s", plan)
	}
	if !strings.Contains(plan, "dev-a.yaml") || !strings.Contains(plan, "prd-a.yaml") {
		t.Fatalf("expected a.yaml outputs, got:\n%s", plan)
	}
	if !strings.Contains(plan, "dev-b.yaml") || !strings.Contains(plan, "prd-b.yaml") {
		t.Fatalf("expected b.yaml outputs, got:\n%s", plan)
	}
}

func TestPlan_SpecLess_NoTarget_NotesUnstableFilename(t *testing.T) {
	t.Parallel()

	plan, err := Plan(Config{IsSpecFile: false, OverrideTemplate: "t.yaml"})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if !strings.Contains(plan, "Notes:") {
		t.Fatalf("expected notes section, got:\n%s", plan)
	}
}
