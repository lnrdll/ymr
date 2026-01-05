package cmd

import (
	"os"

	"github.com/lnrdll/ymr/internal/app"

	"github.com/spf13/cobra"
)

var cfg = app.Config{}

var rootCmd = &cobra.Command{
	Use:   "ymr [command]",
	Short: "A flexible, spec-driven YAML templating tool.",
	Long: `
	    ██ ██ ███▄███▄ ████▄ 
	    ██▄██ ██ ██ ██ ██ ▀▀ 
	     ▀██▀ ██ ██ ██ ██    (ya·mr)
	      ██
	    ▀▀▀
'ymr' is a lightweight, spec-driven YAML template tool.`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
		os.Exit(1)
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {}
