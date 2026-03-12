package cmd

import (
	"github.com/lnrdll/ymr/internal/app"
	"github.com/lnrdll/ymr/internal/logger"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:     "run [flags]",
	Short:   "Processes templates and generates output files.",
	Aliases: []string{"build", "render"},
	Long: `The 'run' command processes YAML templates based on a spec file and
generates output files. It supports overriding parameters and targets
via CLI flags.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		logger.Setup(cfg.Debug)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg.IsSpecFile = cmd.Flags().Changed("spec")
		return app.NewRunCommand(cfg).Execute()
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.PersistentFlags().StringVarP(
		&cfg.SpecFile,
		"spec",
		"s",
		"",
		"Path to a spec file or directory. If used, this flag cannot be empty. (use '.' for current directory)",
	)

	runCmd.PersistentFlags().StringVarP(
		&cfg.OverrideTemplate,
		"template",
		"T",
		"",
		"A single template file/URL. Required in spec-less mode",
	)

	runCmd.PersistentFlags().StringVarP(
		&cfg.OutputDir,
		"output",
		"o",
		"",
		"Output directory for rendered files (use '-' for stdout)",
	)

	runCmd.PersistentFlags().StringSliceVarP(
		&cfg.OverrideParams,
		"param",
		"p",
		nil,
		"Override a parameter (key=value). Can be used multiple times",
	)

	runCmd.PersistentFlags().StringSliceVar(
		&cfg.OverrideParamYAML,
		"param-yaml",
		nil,
		"Override parameters from an inline YAML/JSON mapping. Can be used multiple times",
	)

	runCmd.PersistentFlags().StringSliceVar(
		&cfg.OverrideParamFiles,
		"param-file",
		nil,
		"Override parameters from a YAML/JSON file/URL. Can be used multiple times",
	)

	runCmd.PersistentFlags().StringSliceVarP(
		&cfg.OverrideTargets,
		"target",
		"t",
		nil,
		"Override which targets to render. Can be used multiple times",
	)

	runCmd.PersistentFlags().StringVar(
		&cfg.GithubToken,
		"token",
		"",
		"GitHub token for accessing private repositories (or use GITHUB_TOKEN env var)",
	)

	runCmd.PersistentFlags().StringVar(
		&cfg.ValidationFile,
		"validation",
		"",
		"Path to a validation file. If provided, this will override any validations in the spec file",
	)

	runCmd.PersistentFlags().BoolVarP(
		&cfg.Debug,
		"debug",
		"d",
		false,
		"Enable debug logging",
	)

	runCmd.PersistentFlags().BoolVar(
		&cfg.Strict,
		"strict",
		false,
		"Fail if any template/target render step errors",
	)

	runCmd.PersistentFlags().BoolVar(
		&cfg.ValidateOnly,
		"validate",
		false,
		"Validate spec/params/templates without writing output",
	)
}
