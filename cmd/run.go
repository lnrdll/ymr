package cmd

import (
	"log"
	"ymr/internal/app"
	"ymr/internal/logger"

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
	Run: func(cmd *cobra.Command, args []string) {
		cfg.IsSpecFile = cmd.Flags().Changed("spec")

		if !cfg.IsSpecFile {
			// Spec-less mode
			cfg.SpecFile = ""
			if cfg.OverrideTemplate == "" {
				log.Fatalf("\nError: in spec-less mode (no -s flag), --template (-t) is required.\n")
			}
			if len(cfg.OverrideParams) == 0 {
				log.Fatalf("\nError: in spec-less mode (no -s flag), at least one --param is required.\n")
			}
		}

		if err := app.Run(cfg); err != nil {
			log.Fatalf("Error: %v\n", err)
		}
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
		"A single template file/URL. Required in spec-less mode.",
	)

	runCmd.PersistentFlags().StringVarP(
		&cfg.OutputDir,
		"output",
		"o",
		"",
		"Output directory for rendered files. (use '-' for stdout)",
	)

	runCmd.PersistentFlags().StringSliceVarP(
		&cfg.OverrideParams,
		"param",
		"p",
		nil,
		"Override a parameter (key=value). Can be used multiple times.",
	)

	runCmd.PersistentFlags().StringSliceVarP(
		&cfg.OverrideTargets,
		"target",
		"t",
		nil,
		"Override which targets to render. Can be used multiple times.",
	)

	runCmd.PersistentFlags().StringVar(
		&cfg.GithubToken,
		"token",
		"",
		"GitHub token for accessing private repositories (or use GITHUB_TOKEN env var)",
	)

	runCmd.PersistentFlags().StringVar(
		&cfg.PolicyFile,
		"policy",
		"",
		"Path to a policy file. If provided, this will override any policies in the spec file.",
	)

	runCmd.PersistentFlags().BoolVar(
		&cfg.Debug,
		"debug",
		false,
		"Enable debug logging",
	)
}
