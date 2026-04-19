package cmd

import (
	"fmt"

	"github.com/janklabs/obscuro/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the current version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "obscuro %s\n", version.Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
