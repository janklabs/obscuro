package cmd

import (
	"fmt"
	"os"

	"github.com/janklabs/obscuro/internal/crypto"
	"github.com/janklabs/obscuro/internal/store"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var secretValue string

var setCmd = &cobra.Command{
	Use:   "set KEY",
	Short: "Set an encrypted secret",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		key, err := authenticate()
		if err != nil {
			return err
		}

		var value string
		if secretValue != "" {
			value = secretValue
		} else {
			tty, err := openTTY()
			if err != nil {
				return fmt.Errorf("cannot open terminal for secret prompt: %w", err)
			}
			defer tty.Close()

			fmt.Fprint(tty, "Enter secret value: ")
			raw, err := term.ReadPassword(int(tty.Fd()))
			fmt.Fprintln(tty)
			if err != nil {
				return fmt.Errorf("reading secret: %w", err)
			}
			value = string(raw)
		}

		if len(value) == 0 {
			return fmt.Errorf("secret value cannot be empty")
		}

		encrypted, err := crypto.Encrypt(key, []byte(value))
		if err != nil {
			return err
		}

		secrets, err := store.LoadSecrets()
		if err != nil {
			return err
		}
		secrets[name] = encrypted
		if err := store.SaveSecrets(secrets); err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "Secret '%s' saved.\n", name)
		return nil
	},
}

func init() {
	setCmd.Flags().StringVar(&secretValue, "value", "", "secret value (avoids interactive prompt)")
	rootCmd.AddCommand(setCmd)
}
