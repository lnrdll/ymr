package cmd

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"text/template"

	"github.com/spf13/cobra"
)

type SpecData struct {
	Templates     []string
	Targets       []string
	isBoilerplate bool
}

const specTemplate = `# A list of templates to process.
# Paths are relative to the location of this spec.yaml.
templates:
{{- range .Templates}}
  - {{.}}
{{- else}}
  - base/service.yaml
  - base/configmap.yaml
{{- end}}

# A simple list of target environments.
targetIds:
{{- range .Targets}}
  - {{.}}
{{- else}}
  - dev
  - prd
{{- end}}

# A list of parameter sets.
parameters:
{{- if .Targets}}
  # --- Shared values for all provided targets ---
  - values:
      name: "myapp-name"
    targetId:
{{- range .Targets}}
      - {{.}}
{{- end}}

{{- range .Targets}}
  # --- Specific values for target '{{.}}' ---
  - values:
      foo: bar
    targetId:
      - {{.}}
{{- end}}
{{- else}}
  # --- Shared values ---
  - values:
      name: "myapp-name"
    targetId: # Which targets this value set applies to
      - dev
      - prd

  # --- Dev-specific values ---
  - values:
      minScale: 1
    targetId:
      - dev

  # --- Prod-specific values ---
  - values:
      minScale: 3
      maxScale: 10
    targetId:
      - prd
{{end}}
`

var (
	customTemplates []string
	customTargets   []string
)

var initCmd = &cobra.Command{
	Use:     "init",
	Short:   "Generates a boilerplate spec.yaml file",
	Aliases: []string{"initialize", "initiliase", "create"},
	Long: `Generates a boilerplate spec.yaml file in the current directory.

This file provides a starting point for defining your templates,
target IDs, and parameters.`,
	RunE: runInitE,
}

func init() {
	initCmd.Flags().StringSliceVar(
		&customTemplates,
		"templates",
		nil,
		"A comma-separated list of template paths (e.g., 'base/service.yaml,base/configmap.yaml')",
	)

	initCmd.Flags().StringSliceVar(
		&customTargets,
		"targets",
		nil,
		"A comma-separated list of target IDs (e.g., 'dev,prd')",
	)

	rootCmd.AddCommand(initCmd)
}

// runInitE generates a boilerplate spec.yaml file in the current directory.
func runInitE(cmd *cobra.Command, args []string) error {
	specFile := "spec.yaml"

	if _, err := os.Stat(specFile); err == nil {
		return fmt.Errorf("%s already exists in this directory", specFile)
	}

	data := SpecData{
		Templates:     customTemplates,
		Targets:       customTargets,
		isBoilerplate: len(customTemplates) == 0 && len(customTargets) == 0,
	}

	tmpl, err := template.New("spec").Parse(specTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse the template: %w", err)
	}

	var content bytes.Buffer
	if err := tmpl.Execute(&content, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	if err := os.WriteFile(specFile, content.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write spec file '%s': %w", specFile, err)
	}

	fmt.Printf("Generated boilerplate file: %s\n", specFile)
	return nil
}
