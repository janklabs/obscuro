package cmd

import (
	"fmt"
	"os"

	"github.com/janklabs/obscuro/internal/crypto"
	"github.com/janklabs/obscuro/internal/keychain"
	"github.com/janklabs/obscuro/internal/store"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage keychain password storage",
}

var authStoreCmd = &cobra.Command{
	Use:   "store",
	Short: "Store the master password in the OS keychain",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !store.IsInitialized() {
			return fmt.Errorf("not initialized — run 'obscuro init' first")
		}

		cfg, err := store.LoadConfig()
		if err != nil {
			return err
		}

		salt, err := cfg.DecodeSalt()
		if err != nil {
			return fmt.Errorf("decoding salt: %w", err)
		}

		// Get and verify password before storing
		pw, err := getPassword("Enter master password: ", cfg.Salt)
		if err != nil {
			return err
		}

		key := crypto.DeriveKey(pw, salt)
		if !crypto.VerifyKey(key, cfg.VerificationToken) {
			return fmt.Errorf("incorrect password")
		}

		if err := keychain.Store(cfg.Salt, pw); err != nil {
			return fmt.Errorf("storing in keychain: %w", err)
		}

		fmt.Fprintln(os.Stderr, "Password stored in OS keychain.")
		return nil
	},
}

var authClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Remove the master password from the OS keychain",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !store.IsInitialized() {
			return fmt.Errorf("not initialized — run 'obscuro init' first")
		}

		cfg, err := store.LoadConfig()
		if err != nil {
			return err
		}

		if err := keychain.Delete(cfg.Salt); err != nil {
			return fmt.Errorf("removing from keychain: %w", err)
		}

		fmt.Fprintln(os.Stderr, "Password removed from OS keychain.")
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check if the master password is stored in the OS keychain",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !store.IsInitialized() {
			return fmt.Errorf("not initialized — run 'obscuro init' first")
		}

		cfg, err := store.LoadConfig()
		if err != nil {
			return err
		}

		if keychain.HasEntry(cfg.Salt) {
			fmt.Fprintln(Stdout, "Keychain: password stored")
		} else {
			fmt.Fprintln(Stdout, "Keychain: no password stored")
		}
		return nil
	},
}

func init() {
	authCmd.AddCommand(authStoreCmd)
	authCmd.AddCommand(authClearCmd)
	authCmd.AddCommand(authStatusCmd)
	rootCmd.AddCommand(authCmd)
}
