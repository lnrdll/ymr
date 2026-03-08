package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func prepareExecutionConfig(cmd *cobra.Command) error {
	cfg.IsSpecFile = cmd.Flags().Changed("spec")
	if cfg.IsSpecFile {
		return nil
	}

	cfg.SpecFile = ""
	if cfg.OverrideTemplate == "" {
		return fmt.Errorf("in spec-less mode (no -s flag), --template (-T) is required")
	}
	if len(cfg.OverrideParams) == 0 && len(cfg.OverrideParamFiles) == 0 && len(cfg.OverrideParamYAML) == 0 {
		return fmt.Errorf("in spec-less mode (no -s flag), at least one of --param, --param-file, or --param-yaml is required")
	}

	return nil
}

func registerSharedFlags(command *cobra.Command, includeOutput bool, targetHelp string, strictHelp string) {
	command.PersistentFlags().StringVarP(
		&cfg.SpecFile,
		"spec",
		"s",
		"",
		"Path to a spec file or directory. If used, this flag cannot be empty. (use '.' for current directory)",
	)

	command.PersistentFlags().StringVarP(
		&cfg.OverrideTemplate,
		"template",
		"T",
		"",
		"A single template file/URL. Required in spec-less mode",
	)

	if includeOutput {
		command.PersistentFlags().StringVarP(
			&cfg.OutputDir,
			"output",
			"o",
			"",
			"Output directory for rendered files (use '-' for stdout)",
		)
	}

	command.PersistentFlags().StringSliceVarP(
		&cfg.OverrideParams,
		"param",
		"p",
		nil,
		"Override a parameter (key=value). Can be used multiple times",
	)

	command.PersistentFlags().StringSliceVar(
		&cfg.OverrideParamYAML,
		"param-yaml",
		nil,
		"Override parameters from an inline YAML/JSON mapping. Can be used multiple times",
	)

	command.PersistentFlags().StringSliceVar(
		&cfg.OverrideParamFiles,
		"param-file",
		nil,
		"Override parameters from a YAML/JSON file/URL. Can be used multiple times",
	)

	command.PersistentFlags().StringSliceVarP(
		&cfg.OverrideTargets,
		"target",
		"t",
		nil,
		targetHelp,
	)

	command.PersistentFlags().StringVar(
		&cfg.GithubToken,
		"token",
		"",
		"GitHub token for accessing private repositories (or use GITHUB_TOKEN env var)",
	)

	command.PersistentFlags().StringVar(
		&cfg.ValidationFile,
		"validation",
		"",
		"Path to a validation file. If provided, this will override any validations in the spec file",
	)

	command.PersistentFlags().BoolVarP(
		&cfg.Debug,
		"debug",
		"d",
		false,
		"Enable debug logging",
	)

	command.PersistentFlags().BoolVar(
		&cfg.Strict,
		"strict",
		false,
		strictHelp,
	)
}
