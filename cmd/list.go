package cmd

import (
	"fmt"

	"github.com/janklabs/obscuro/internal/store"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all secret keys",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !store.IsInitialized() {
			return fmt.Errorf("not initialized — run 'obscuro init' first")
		}

		secrets, err := store.LoadSecrets()
		if err != nil {
			return err
		}

		keys := store.ListKeys(secrets)
		if len(keys) == 0 {
			fmt.Fprintln(Stdout, "No secrets stored.")
			return nil
		}

		for _, k := range keys {
			fmt.Fprintln(Stdout, k)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
