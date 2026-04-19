package cmd

import (
	"fmt"

	"github.com/janklabs/obscuro/internal/crypto"
	"github.com/janklabs/obscuro/internal/store"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get KEY",
	Short: "Decrypt and print a secret value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		key, err := authenticate()
		if err != nil {
			return err
		}

		secrets, err := store.LoadSecrets()
		if err != nil {
			return err
		}

		encrypted, ok := secrets[name]
		if !ok {
			return fmt.Errorf("secret '%s' not found", name)
		}

		plaintext, err := crypto.Decrypt(key, encrypted)
		if err != nil {
			return fmt.Errorf("decrypting secret: %w", err)
		}

		cmd.Print(string(plaintext))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
}
