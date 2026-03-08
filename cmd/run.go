package cmd

import (
	"log"

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
	Run: func(cmd *cobra.Command, args []string) {
		if err := prepareExecutionConfig(cmd); err != nil {
			log.Fatalf("\nError: %v\n", err)
		}

		if err := app.NewRunCommand(cfg).Execute(); err != nil {
			log.Fatalf("Error: %v\n", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	registerSharedFlags(
		runCmd,
		true,
		"Override which targets to render. Can be used multiple times",
		"Fail if any template/target render step errors",
	)

}
