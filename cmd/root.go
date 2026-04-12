package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/lnrdll/ymr/internal/app"

	"github.com/spf13/cobra"
)

var cfg = app.Config{}

var rootCmd = &cobra.Command{
	Use:           "ymr [command]",
	Short:         "A flexible, spec-driven YAML templating tool.",
	SilenceUsage:  true,
	SilenceErrors: true,
	Long: `
	    ██ ██ ███▄███▄ ████▄ 
	    ██▄██ ██ ██ ██ ██ ▀▀ 
	     ▀██▀ ██ ██ ██ ██    (ya·mr)
	      ██
	    ▀▀▀
'ymr' is a lightweight, spec-driven YAML template tool.`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
		os.Exit(0)
	},
}

func formatCLIError(err error) string {
	return fmt.Sprintf("ERROR %s", err)
}

func escapeGitHubActionsMessage(msg string) string {
	msg = strings.ReplaceAll(msg, "%", "%25")
	msg = strings.ReplaceAll(msg, "\r", "%0D")
	msg = strings.ReplaceAll(msg, "\n", "%0A")
	return msg
}

func formatGitHubActionsError(err error) string {
	return fmt.Sprintf("::error::%s", escapeGitHubActionsMessage(err.Error()))
}

func printCLIError(err error) {
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		_, _ = fmt.Fprintln(os.Stderr, formatGitHubActionsError(err))
	}
	_, _ = fmt.Fprintln(os.Stderr, formatCLIError(err))
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		printCLIError(err)
		os.Exit(1)
	}
}

func init() {}
