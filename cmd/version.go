package cmd

import (
	"fmt"

	"github.com/lnrdll/ymr/internal/buildinfo"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the ymr version",
	Run: func(cmd *cobra.Command, args []string) {
		if buildinfo.Commit != "none" && buildinfo.Date != "unknown" {
			cmd.Println(fmt.Sprintf("ymr %s (%s %s)", buildinfo.Version, buildinfo.Commit, buildinfo.Date))
			return
		}

		cmd.Println(fmt.Sprintf("ymr %s", buildinfo.Version))
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
