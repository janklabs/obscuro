package cmd

import (
	"fmt"
	"os"

	"github.com/janklabs/obscuro/internal/crypto"
	"github.com/janklabs/obscuro/internal/store"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize obscuro in the current directory",
	Long:  "Creates the .obscuro directory and sets up encryption with a master password.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if store.IsInitialized() {
			return fmt.Errorf(".obscuro already initialized in this directory")
		}

		var pw string

		if password != "" {
			pw = password
		} else {
			fmt.Fprint(os.Stderr, "Enter master password: ")
			pw1, err := term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Fprintln(os.Stderr)
			if err != nil {
				return fmt.Errorf("reading password: %w", err)
			}
			if len(pw1) == 0 {
				return fmt.Errorf("password cannot be empty")
			}

			fmt.Fprint(os.Stderr, "Confirm master password: ")
			pw2, err := term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Fprintln(os.Stderr)
			if err != nil {
				return fmt.Errorf("reading password: %w", err)
			}

			if string(pw1) != string(pw2) {
				return fmt.Errorf("passwords do not match")
			}
			pw = string(pw1)
		}

		salt, err := crypto.GenerateSalt()
		if err != nil {
			return err
		}

		key := crypto.DeriveKey(pw, salt)
		token, err := crypto.CreateVerificationToken(key)
		if err != nil {
			return err
		}

		if err := store.Init(salt, token); err != nil {
			return err
		}

		fmt.Fprintln(os.Stderr, "Initialized .obscuro successfully.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
