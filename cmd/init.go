package cmd

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/janklabs/obscuro/internal/crypto"
	"github.com/janklabs/obscuro/internal/keychain"
	"github.com/janklabs/obscuro/internal/pwfile"
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

		// Offer to store the password in a backend (only for interactive sessions).
		// The cfg is constructed in-memory here rather than re-read from disk so the
		// selector's post-choice SaveConfig writes the freshly-init'd config exactly
		// once. Fields mirror what store.Init just wrote; SchemaVersion is set to 2
		// so LoadConfig-based readers see the current schema even before SaveConfig.
		if password == "" {
			saltB64 := base64.StdEncoding.EncodeToString(salt)
			cfg := &store.Config{
				Salt:              saltB64,
				VerificationToken: token,
				SchemaVersion:     2,
			}
			offerBackendSelector(pw, saltB64, cfg)
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

// offerBackendSelector asks the user whether to store the master password, then
// delegates backend selection to runBackendSelectorFn (the TUI). On success it
// writes cfg.PasswordBackend and persists cfg via store.SaveConfig. This is a
// fire-and-forget helper: init exits 0 regardless of any error here — the user
// can always run `obscuro auth store` later.
func offerBackendSelector(pw, salt string, cfg *store.Config) {
	answer, hasTTY := offerKeychainConfirmFn("Store password for future use? [Y/n] ")
	if !hasTTY {
		return
	}
	if answer != "" && answer != "y" && answer != "yes" {
		return
	}

	statuses := detectBackends(*cfg)
	choice, err := runBackendSelectorFn(statuses, false)
	if err != nil {
		if errors.Is(err, ErrNonInteractive) || errors.Is(err, ErrCancelled) {
			fmt.Fprintln(os.Stderr, "Skipping backend setup; run 'obscuro auth store' later.")
			return
		}
		fmt.Fprintf(os.Stderr, "backend selector: %v\n", err)
		return
	}

	switch choice.Kind {
	case BackendKeychain:
		if err := keychain.Store(salt, pw); err != nil {
			fmt.Fprintf(os.Stderr, "storing password in keychain: %v\n", err)
			return
		}
	case BackendFile:
		if err := pwfile.Write(salt, pw); err != nil {
			fmt.Fprintf(os.Stderr, "writing password file: %v\n", err)
			return
		}
	}

	cfg.PasswordBackend = string(choice.Kind)
	if err := store.SaveConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "saving config: %v\n", err)
	}
	fmt.Fprintf(os.Stderr, "Password stored via %s.\n", choice.Kind)
}

func init() {
	rootCmd.AddCommand(initCmd)
}
