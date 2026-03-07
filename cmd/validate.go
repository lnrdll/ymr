package cmd

import (
	"log"

	"github.com/lnrdll/ymr/internal/app"
	"github.com/lnrdll/ymr/internal/logger"

	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate [flags]",
	Short: "Validate spec/params/templates without writing output.",
	Long: `The 'validate' command loads the spec (or runs in spec-less mode), applies
parameter and target overrides, runs validation rules, and checks templates can be
loaded and parsed.

It does not write rendered output files.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		logger.Setup(cfg.Debug)
	},
	Run: func(cmd *cobra.Command, args []string) {
		cfg.IsSpecFile = cmd.Flags().Changed("spec")

		if !cfg.IsSpecFile {
			cfg.SpecFile = ""
			if cfg.OverrideTemplate == "" {
				log.Fatalf("\nError: in spec-less mode (no -s flag), --template (-T) is required.\n")
			}
			if len(cfg.OverrideParams) == 0 && len(cfg.OverrideParamFiles) == 0 && len(cfg.OverrideParamYAML) == 0 {
				log.Fatalf("\nError: in spec-less mode (no -s flag), at least one of --param, --param-file, or --param-yaml is required.\n")
			}
		}

		if err := app.NewValidateCommand(cfg).Execute(); err != nil {
			log.Fatalf("Error: %v\n", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)

	validateCmd.PersistentFlags().StringVarP(
		&cfg.SpecFile,
		"spec",
		"s",
		"",
		"Path to a spec file or directory. If used, this flag cannot be empty. (use '.' for current directory)",
	)

	validateCmd.PersistentFlags().StringVarP(
		&cfg.OverrideTemplate,
		"template",
		"T",
		"",
		"A single template file/URL. Required in spec-less mode",
	)

	validateCmd.PersistentFlags().StringSliceVarP(
		&cfg.OverrideParams,
		"param",
		"p",
		nil,
		"Override a parameter (key=value). Can be used multiple times",
	)

	validateCmd.PersistentFlags().StringSliceVar(
		&cfg.OverrideParamYAML,
		"param-yaml",
		nil,
		"Override parameters from an inline YAML/JSON mapping. Can be used multiple times",
	)

	validateCmd.PersistentFlags().StringSliceVar(
		&cfg.OverrideParamFiles,
		"param-file",
		nil,
		"Override parameters from a YAML/JSON file/URL. Can be used multiple times",
	)

	validateCmd.PersistentFlags().StringSliceVarP(
		&cfg.OverrideTargets,
		"target",
		"t",
		nil,
		"Override which targets to validate. Can be used multiple times",
	)

	validateCmd.PersistentFlags().StringVar(
		&cfg.GithubToken,
		"token",
		"",
		"GitHub token for accessing private repositories (or use GITHUB_TOKEN env var)",
	)

	validateCmd.PersistentFlags().StringVar(
		&cfg.ValidationFile,
		"validation",
		"",
		"Path to a validation file. If provided, this will override any validations in the spec file",
	)

	validateCmd.PersistentFlags().BoolVarP(
		&cfg.Debug,
		"debug",
		"d",
		false,
		"Enable debug logging",
	)

	validateCmd.PersistentFlags().BoolVar(
		&cfg.Strict,
		"strict",
		false,
		"Fail if any template/target validation step errors",
	)
}
