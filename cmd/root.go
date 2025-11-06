package cmd

import (
	"os"
	"ymr/internal/app"

	"github.com/spf13/cobra"
)

var cfg = app.Config{}

var rootCmd = &cobra.Command{
	Use:   "ymr [command]",
	Short: "A flexible, spec-driven YAML templating tool.",
	Long: `
██╗   ██╗███╗   ███╗██████╗ 
╚██╗ ██╔╝████╗ ████║██╔══██╗
 ╚████╔╝ ██╔████╔██║██████╔╝
  ╚██╔╝  ██║╚██╔╝██║██╔══██╗
   ██║   ██║ ╚═╝ ██║██║  ██║
   ╚═╝   ╚═╝     ╚═╝╚═╝  ╚═╝ (ya·mr)

'ymr' is a CLI tool that generates YAML files from a spec and one
or more templates. It replaces values in the templates marked
with '# from-param: <name>' or '# from-param-merge: <name>'
with values from the spec.`,
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

func init() {
}
