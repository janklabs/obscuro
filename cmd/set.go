package cmd

import (
	"fmt"
	"os"

	"github.com/janklabs/obscuro/internal/crypto"
	"github.com/janklabs/obscuro/internal/store"
	"github.com/spf13/cobra"
)

var secretValue string
var secretValueFile string

var setCmd = &cobra.Command{
	Use:          "set KEY",
	Short:        "Set an encrypted secret",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		key, err := authenticate()
		if err != nil {
			return err
		}

		var value string
		if secretValue != "" {
			value = secretValue
		} else if secretValueFile != "" {
			v, err := readSecretFile(secretValueFile)
			if err != nil {
				return fmt.Errorf("reading value file: %w", err)
			}
			value = v
		} else {
			fmt.Fprintf(os.Stderr, "Enter the secret value for '%s'.\n", name)
			v, err := promptPasswordFn("Enter secret value: ")
			if err != nil {
				return fmt.Errorf("reading secret: %w", err)
			}
			value = v
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

		fmt.Fprintf(os.Stderr, "Secret '%s' was set.\n", name)
		return nil
	},
}

func init() {
	setCmd.Flags().StringVar(&secretValue, "value", "", "secret value (avoids interactive prompt; visible in process list)")
	setCmd.Flags().StringVar(&secretValueFile, "value-file", "", "read secret value from file (or - for stdin)")
	rootCmd.AddCommand(setCmd)
}
