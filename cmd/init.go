package cmd

import (
	"bytes"
	"fmt"
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
  - service.yaml
{{- end}}

# A simple list of target environments. Only targets
# listed here can be referenced in parameters or in
# validations.
targetIds:
{{- range .Targets}}
  - {{.}}
{{- else}}
  - dev
  - prd
{{- end}}

# Validations are expressed using CEL: https://cel.dev/
validations:
  - rule: "params.minScale >= 1"
    message: "minScale must be greater than or equal to 1"
    targetId: ["prd"]

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
      foo: bar
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
	RunE: runInit,
}

var forceInit bool

func init() {
	initCmd.Flags().BoolVar(
		&forceInit,
		"force",
		false,
		"Overwrite existing spec.yaml",
	)

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

func writeSpecFile(specFile string, content []byte, force bool) error {
	if !force {
		if _, err := os.Stat(specFile); err == nil {
			return fmt.Errorf("%s already exists in this directory", specFile)
		} else if !os.IsNotExist(err) {
			return err
		}
	}

	return os.WriteFile(specFile, content, 0644)
}

// runInit generates a boilerplate spec.yaml file in the current directory.
func runInit(cmd *cobra.Command, args []string) error {
	specFile := "spec.yaml"

	specData.isBoilerplate = len(specData.Templates) == 0 && len(specData.Targets) == 0

	tmpl, err := template.New("spec").Parse(specTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse init template: %w", err)
	}

	var content bytes.Buffer
	if err := tmpl.Execute(&content, specData); err != nil {
		return fmt.Errorf("failed to execute init template: %w", err)
	}

	if err := writeSpecFile(specFile, content.Bytes(), forceInit); err != nil {
		return fmt.Errorf("failed to write spec file: %w", err)
	}

	fmt.Printf("Generated boilerplate file: %s\n", specFile)
	return nil
}
