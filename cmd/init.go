package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

const boilerplateSpec = `# A list of templates to process.
# Paths are relative to the location of this spec.yaml.
templates:
  - base/service.yaml
  - base/configmap.yaml

# A simple list of target environments.
targetIds:
  - dev
  - prd

# A list of parameter sets.
parameters:
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

	err := os.WriteFile(specFile, []byte(boilerplateSpec), 0644)
	if err != nil {
		slog.Debug(fmt.Sprintf("Error writing spec file '%s': %v", specFile, err))
		os.Exit(1)
	}

	fmt.Printf("Generated boilerplate file: %s\n", specFile)
}
