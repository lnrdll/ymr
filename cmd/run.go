package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"ymr/internal/app"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [flags]",
	Short: "Processes templates and generates output files.",
	Long: `The 'run' command processes YAML templates based on a spec file and
generates output files. It supports overriding parameters and targets
via CLI flags.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var programLevel = slog.LevelWarn
		if cfg.Debug {
			programLevel = slog.LevelDebug
		}

		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level:     programLevel,
			AddSource: true,
		}))
		slog.SetDefault(logger)
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if err := app.Run(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.PersistentFlags().StringVarP(
		&cfg.SpecFile,
		"file",
		"f",
		"spec.yaml",
		"Source path (local, dir, file, http url, or github). Defaults to ./spec.yaml)",
	)

	runCmd.PersistentFlags().StringVarP(
		&cfg.OverrideTemplate,
		"template",
		"t",
		"",
		"A single template file/URL to override the 'templates' list in the spec",
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

	runCmd.PersistentFlags().StringSliceVar(
		&cfg.OverrideTargets,
		"target",
		nil,
		"Override which targets to render. Can be used multiple times.",
	)

	runCmd.PersistentFlags().StringVar(
		&cfg.GithubToken,
		"token",
		"",
		"GitHub token for accessing private repositories (or use GITHUB_TOKEN env var)",
	)

	runCmd.PersistentFlags().BoolVar(
		&cfg.Debug,
		"debug",
		false,
		"Enable debug logging",
	)
}
