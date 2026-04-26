package cmd

import (
	"fmt"

	"github.com/janklabs/obscuro/internal/store"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:     "remove KEY",
	Aliases: []string{"rm"},
	Short:   "Remove a secret",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		if err := store.DeleteSecret(name); err != nil {
			return err
		}

		fmt.Fprintf(cmd.ErrOrStderr(), "Secret '%s' removed.\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
