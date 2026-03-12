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

func TestRun_SpecLess_ParamFile_SupportsMapsAndLists(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	templatePath := filepath.Join(workDir, "t.yaml")
	paramsPath := filepath.Join(workDir, "params.yaml")
	outDir := filepath.Join(workDir, "out")

	err := os.WriteFile(templatePath, []byte(strings.TrimSpace(`
metadata:
  name: default # from-param: {{ .name }}
  labels: # from-param: {{ .labels }}
    foo: bar
spec:
  ports: [1] # from-param: {{ .ports }}
`)+"\n"), 0644)
	if err != nil {
		t.Fatalf("write template: %v", err)
	}

	err = os.WriteFile(paramsPath, []byte(strings.TrimSpace(`
name: myapp
labels:
  env: dev
ports:
  - 80
  - 443
`)+"\n"), 0644)
	if err != nil {
		t.Fatalf("write params: %v", err)
	}

	cfg := Config{
		IsSpecFile:         false,
		OverrideTemplate:   templatePath,
		OverrideParamFiles: []string{paramsPath},
		OverrideTargets:    []string{"dev"},
		OutputDir:          outDir,
	}

	if err := Run(cfg); err != nil {
		t.Fatalf("Run: %v", err)
	}

	outFile := filepath.Join(outDir, "dev-t.yaml")
	b, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output %s: %v", outFile, err)
	}
	out := string(b)
	if !strings.Contains(out, "name: myapp") {
		t.Fatalf("expected name override; got:\n%s", out)
	}
	if !strings.Contains(out, "env: dev") {
		t.Fatalf("expected labels map override; got:\n%s", out)
	}
	if !strings.Contains(out, "80") || !strings.Contains(out, "443") {
		t.Fatalf("expected ports list override; got:\n%s", out)
	}
}

func TestResolveParamsForTarget_DoesNotMutateLookup(t *testing.T) {
	t.Parallel()

	lookup := map[string]map[string]any{
		"dev": {
			"name": "base",
		},
	}
	overrides := map[string]any{"name": "override", "newKey": true}

	resolved := resolveParamsForTarget(lookup, "dev", overrides)

	if resolved["name"] != "override" {
		t.Fatalf("expected override to apply, got %#v", resolved)
	}
	if resolved["newKey"] != true {
		t.Fatalf("expected new override key, got %#v", resolved)
	}
	if lookup["dev"]["name"] != "base" {
		t.Fatalf("expected lookup to stay immutable, got %#v", lookup["dev"])
	}
}

func TestRun_SpecLess_FailsWithoutTemplate(t *testing.T) {
	t.Parallel()

	cfg := Config{
		IsSpecFile:      false,
		OverrideParams:  []string{"name=myapp"},
		OverrideTargets: []string{"dev"},
	}

	err := Run(cfg)
	if err == nil {
		t.Fatalf("expected error for missing template in spec-less mode")
	}
}

func TestRun_SpecLess_FailsWithoutParams(t *testing.T) {
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
	}

	err = Run(cfg)
	if err == nil {
		t.Fatalf("expected error for missing params in spec-less mode")
	}
}
