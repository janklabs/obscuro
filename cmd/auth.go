package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/janklabs/obscuro/internal/crypto"
	"github.com/janklabs/obscuro/internal/keychain"
	"github.com/janklabs/obscuro/internal/pwfile"
	"github.com/janklabs/obscuro/internal/store"
	"github.com/spf13/cobra"
)

var (
	authVerbose bool
	authBackend string
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage keychain password storage",
}

var authStoreCmd = &cobra.Command{
	Use:          "store",
	Short:        "Store the master password in the OS keychain",
	SilenceUsage: true,
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

		statuses := detectBackends(*cfg)

		var choice backendChoice
		if authBackend != "" {
			var found bool
			for _, s := range statuses {
				if string(s.Kind) == authBackend {
					found = true
					if !s.Available {
						return &exitErr{
							code: 3,
							msg:  fmt.Sprintf("backend %q is not available: %s", authBackend, s.Reason),
						}
					}
					choice = backendChoice{Kind: s.Kind, Verbose: authVerbose}
					break
				}
			}
			if !found {
				return &exitErr{
					code: 3,
					msg:  fmt.Sprintf("unknown backend %q — valid options: keychain, file", authBackend),
				}
			}
		} else {
			var selErr error
			choice, selErr = runBackendSelectorFn(statuses, authVerbose)
			if selErr != nil {
				if errors.Is(selErr, ErrNonInteractive) {
					return &exitErr{code: 2, msg: "not a TTY — use --backend flag"}
				}
				if errors.Is(selErr, ErrCancelled) {
					return &exitErr{code: 2, msg: "cancelled"}
				}
				return selErr
			}
		}

		// Set PasswordBackend in-memory BEFORE getPassword so getPassword's
		// backend-check step sees the just-chosen backend on first run.
		cfg.PasswordBackend = string(choice.Kind)

		pw, err := getPassword("Enter master password: ", *cfg)
		if err != nil {
			return err
		}

		key := crypto.DeriveKey(pw, salt)
		if !crypto.VerifyKey(key, cfg.VerificationToken) {
			return fmt.Errorf("incorrect password — verify your input or re-initialize with 'obscuro init'")
		}

		switch choice.Kind {
		case BackendKeychain:
			if err := keychain.Store(cfg.Salt, pw); err != nil {
				return fmt.Errorf("storing in keychain: %w", err)
			}
		case BackendFile:
			if err := pwfile.Write(cfg.Salt, pw); err != nil {
				return fmt.Errorf("writing password file: %w", err)
			}
		}

		if err := store.SaveConfig(cfg); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		fmt.Fprintf(os.Stderr, "Password stored via %s.\n", choice.Kind)
		return nil
	},
}

var authClearCmd = &cobra.Command{
	Use:          "clear",
	Short:        "Remove the master password from the OS keychain",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !store.IsInitialized() {
			return fmt.Errorf("not initialized — run 'obscuro init' first")
		}

		cfg, err := store.LoadConfig()
		if err != nil {
			return err
		}

		if !keychain.HasEntry(cfg.Salt) {
			fmt.Fprintln(cmd.ErrOrStderr(), "no keychain entry found (nothing to clear)")
			return nil
		}

		if err := keychain.Delete(cfg.Salt); err != nil {
			return fmt.Errorf("removing from keychain: %w", err)
		}

		fmt.Fprintln(cmd.ErrOrStderr(), "Password removed from OS keychain.")
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:          "status",
	Short:        "Check OS keychain status",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !store.IsInitialized() {
			return fmt.Errorf("not initialized — run 'obscuro init' first")
		}

		cfg, err := store.LoadConfig()
		if err != nil {
			return err
		}

		backend := cfg.PasswordBackend
		if backend == "" {
			backend = "none"
		}
		fmt.Fprintf(Stdout, "Configured backend: %s\n", backend)

		statuses := detectBackends(*cfg)
		for _, s := range statuses {
			mark := "✗"
			if s.Available {
				mark = "✓"
			}
			fmt.Fprintf(Stdout, "[%s] %s: %s — %s\n", mark, s.Kind, s.Name, s.Reason)
			if authVerbose {
				for _, line := range s.Verbose {
					fmt.Fprintf(Stdout, "    %s\n", line)
				}
			}
		}

		fingerprint := cfg.Salt
		if len(fingerprint) > 8 {
			fingerprint = fingerprint[:8]
		}
		fmt.Fprintf(Stdout, "Salt fingerprint: %s...\n", fingerprint)

		root, err := store.RepoRoot()
		if err != nil {
			return err
		}
		fmt.Fprintf(Stdout, "Repo: %s\n", root)
		return nil
	},
}

func init() {
	authCmd.PersistentFlags().BoolVar(&authVerbose, "verbose", false, "show verbose backend diagnostics")
	authStoreCmd.Flags().StringVar(&authBackend, "backend", "", "select backend non-interactively: keychain or file")

	authCmd.AddCommand(authStoreCmd)
	authCmd.AddCommand(authClearCmd)
	authCmd.AddCommand(authStatusCmd)
	rootCmd.AddCommand(authCmd)
}
