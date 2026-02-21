package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_SpecLess_AllowsTargetOverride(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	templatePath := filepath.Join(workDir, "configmap.yaml")
	outDir := filepath.Join(workDir, "out")

	err := os.WriteFile(templatePath, []byte(strings.TrimSpace(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: default # from-param: {{ .name }}
`)+"\n"), 0644)
	if err != nil {
		t.Fatalf("write template: %v", err)
	}

	cfg := Config{
		IsSpecFile:       false,
		OverrideTemplate: templatePath,
		OverrideParams:   []string{"name=myapp"},
		OverrideTargets:  []string{"dev"},
		OutputDir:        outDir,
	}

	if err := Run(cfg); err != nil {
		t.Fatalf("Run: %v", err)
	}

	outFile := filepath.Join(outDir, "dev-configmap.yaml")
	b, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output %s: %v", outFile, err)
	}

	out := string(b)
	if !strings.Contains(out, "name: myapp") {
		t.Fatalf("expected rendered param in output; got:\n%s", out)
	}
	if strings.Contains(out, "from-param") {
		t.Fatalf("expected directives removed from output; got:\n%s", out)
	}
}

func TestRun_Strict_FailsOnMissingTemplate(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	specPath := filepath.Join(workDir, "spec.yaml")
	outDir := filepath.Join(workDir, "out")

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

	baseCfg := Config{
		IsSpecFile: true,
		SpecFile:   specPath,
		OutputDir:  outDir,
	}

	if err := Run(baseCfg); err != nil {
		t.Fatalf("expected non-strict run to succeed, got: %v", err)
	}

	strictCfg := baseCfg
	strictCfg.Strict = true
	if err := Run(strictCfg); err == nil {
		t.Fatalf("expected strict run to fail")
	}
}
