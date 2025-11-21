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
	Parameters    []string
	isBoilerplate bool
}

var specData = SpecData{}

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
  - targetId:
{{- range .Targets}}
      - {{.}}
{{- end}}
    values:
{{- if $.Parameters}}
{{- range $.Parameters}}
      {{.}}
{{- end}}
{{- else}}
      name: "myapp-name"
{{- end}}

{{- range .Targets}}
  # --- Specific values for target '{{.}}' ---
  - targetId: ["{{.}}"]
    values:
{{- if $.Parameters}}
{{- range $.Parameters}}
      {{.}}
{{- end}}
{{- else}}
      foo: bar
{{- end}}
{{- end}}
{{- else}}
  # --- Shared values ---
  - targetId: ["dev", "prd"] # Which targets this value set applies to
    values:
      name: "myapp-name"

  # --- Dev-specific values ---
  - targetId: ["dev"]
    values:
      minScale: 1

  # --- Prod-specific values ---
  - targetId: ["prd"]
    values:
      minScale: 3
      maxScale: 10
{{end}}
`

var initCmd = &cobra.Command{
	Use:     "init",
	Short:   "Generates a boilerplate spec.yaml file",
	Aliases: []string{"initialize", "initiliase", "create"},
	Long: `Generates a boilerplate spec.yaml file in the current directory.

This file provides a starting point for defining your templates,
target IDs, and parameters.`,
	Run: runInit,
}

func init() {
	initCmd.Flags().StringSliceVar(
		&specData.Templates,
		"templates",
		nil,
		"A comma-separated list of template paths (e.g., 'base/service.yaml,base/configmap.yaml')",
	)

	initCmd.Flags().StringSliceVarP(
		&specData.Targets,
		"target",
		"t",
		nil,
		"A comma-separated list of target IDs (e.g., 'dev,prd')",
	)

	initCmd.PersistentFlags().StringSliceVarP(
		&specData.Parameters,
		"param",
		"p",
		nil,
		"Param values to be configured ('foo: bar'). Can be used multiple times.",
	)

	rootCmd.AddCommand(initCmd)
}

// runInit generates a boilerplate spec.yaml file in the current directory.
func runInit(cmd *cobra.Command, args []string) {
	specFile := "spec.yaml"

	if _, err := os.Stat(specFile); err == nil {
		// TODO
		slog.Error(specFile + "already exists in this directory.")
		os.Exit(1)
	}

	specData.isBoilerplate = len(specData.Templates) == 0 && len(specData.Targets) == 0

	tmpl, err := template.New("spec").Parse(specTemplate)
	if err != nil {
		slog.Error("failed to parse the template", "error", err)
		os.Exit(1)
	}

	var content bytes.Buffer
	if err := tmpl.Execute(&content, specData); err != nil {
		slog.Error("failed to execute template", "error", err)
		os.Exit(1)
	}

	err = os.WriteFile(specFile, content.Bytes(), 0644)
	if err != nil {
		slog.Debug(fmt.Sprintf("Error writing spec file '%s': %v", specFile, err))
		os.Exit(1)
	}

	fmt.Printf("Generated boilerplate file: %s\n", specFile)
}
