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
		if err := prepareExecutionConfig(cmd); err != nil {
			log.Fatalf("\nError: %v\n", err)
		}

		if err := app.NewValidateCommand(cfg).Execute(); err != nil {
			log.Fatalf("Error: %v\n", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
	registerSharedFlags(
		validateCmd,
		false,
		"Override which targets to validate. Can be used multiple times",
		"Fail if any template/target validation step errors",
	)
}
