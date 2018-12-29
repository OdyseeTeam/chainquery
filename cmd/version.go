package cmd

import (
	"github.com/lbryio/chainquery/meta"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Chainquery",
	Long:  `All software has versions. This is Chainquery's`,
	Run: func(cmd *cobra.Command, args []string) {
		println("Semantic Version: ", meta.GetSemVersion())
		println("Version: " + meta.GetVersion())
		println("Version(long): " + meta.GetVersionLong())
	},
}
