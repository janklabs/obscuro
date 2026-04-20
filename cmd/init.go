package cmd

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/janklabs/obscuro/internal/crypto"
	"github.com/janklabs/obscuro/internal/keychain"
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
			tty, err := openTTY()
			if err != nil {
				return fmt.Errorf("cannot open terminal for password prompt: %w", err)
			}
			defer tty.Close()

			fmt.Fprint(tty, "Enter master password: ")
			pw1, err := term.ReadPassword(int(tty.Fd()))
			fmt.Fprintln(tty)
			if err != nil {
				return fmt.Errorf("reading password: %w", err)
			}
			if len(pw1) == 0 {
				return fmt.Errorf("password cannot be empty")
			}

			fmt.Fprint(tty, "Confirm master password: ")
			pw2, err := term.ReadPassword(int(tty.Fd()))
			fmt.Fprintln(tty)
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

		// Offer to store password in OS keychain (only for interactive sessions)
		if password == "" {
			saltB64 := base64.StdEncoding.EncodeToString(salt)
			offerKeychainStore(pw, saltB64)
		}

		return nil
	},
}

// offerKeychainStore prompts the user to store the password in the OS keychain.
// Silently skips if no TTY is available (non-interactive / CI).
func offerKeychainStore(pw, salt string) {
	tty, err := openTTY()
	if err != nil {
		return
	}
	defer tty.Close()

	fmt.Fprint(tty, "Store password in OS keychain? [Y/n] ")
	reader := bufio.NewReader(tty)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	if answer == "" || answer == "y" || answer == "yes" {
		if err := keychain.Store(salt, pw); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not store in keychain: %v\n", err)
			return
		}
		fmt.Fprintln(tty, "Password stored in OS keychain.")
	}
}

func init() {
	rootCmd.AddCommand(initCmd)
}
