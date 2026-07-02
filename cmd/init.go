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
)

// offerKeychainConfirmFn is a seam for testing keychain confirmation prompts.
var offerKeychainConfirmFn = confirmKeychainStore

var initCmd = &cobra.Command{
	Use:          "init",
	Short:        "Initialize encryption for this repo",
	Long:         "Creates the .obscuro directory and sets up encryption with a master password.",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store.IsInitialized() {
			return fmt.Errorf(".obscuro already initialized in this repository")
		}

		var pw string

		if password != "" {
			pw = password
		} else {
			pw1, err := promptPasswordFn("Enter master password: ")
			if err != nil {
				return err
			}
			if len(pw1) == 0 {
				return fmt.Errorf("password cannot be empty")
			}
			pw2, err := promptPasswordFn("Confirm master password: ")
			if err != nil {
				return err
			}
			if pw1 != pw2 {
				return fmt.Errorf("passwords do not match")
			}
			pw = pw1
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

		fmt.Fprintln(os.Stderr, "Initialized .obscuro.")

		// Offer to store password in OS keychain (only for interactive sessions)
		if password == "" {
			saltB64 := base64.StdEncoding.EncodeToString(salt)
			offerKeychainStore(pw, saltB64)
		}

		return nil
	},
}

// confirmKeychainStore prompts the user via TTY to confirm storing password in keychain.
// Returns the user's answer (lowercased, trimmed) and whether a TTY was available.
func confirmKeychainStore(prompt string) (string, bool) {
	tty, err := openTTY()
	if err != nil {
		return "", false
	}
	defer tty.Close()

	fmt.Fprint(tty, prompt)
	reader := bufio.NewReader(tty)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer, true
}

// offerKeychainStore prompts the user to store the password in the OS keychain.
// Silently skips if no TTY is available (non-interactive / CI).
func offerKeychainStore(pw, salt string) {
	if err := keychain.Available(); err != nil {
		fmt.Fprintln(os.Stderr, keychainRemediation().String())
		return
	}

	answer, hasTTY := offerKeychainConfirmFn("Store password in OS keychain? [Y/n] ")
	if !hasTTY {
		return
	}

	if answer == "" || answer == "y" || answer == "yes" {
		if err := keychain.Store(salt, pw); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", keychainRemediation().Error(), err)
			return
		}
		if tty, err := openTTY(); err == nil {
			fmt.Fprintln(tty, "Password stored in OS keychain.")
			tty.Close()
		}
	}
}

func init() {
	rootCmd.AddCommand(initCmd)
}
