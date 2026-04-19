package cmd

import (
	"fmt"
	"os"

	"github.com/janklabs/obscuro/internal/crypto"
	"github.com/janklabs/obscuro/internal/store"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var password string

var rootCmd = &cobra.Command{
	Use:   "obscuro",
	Short: "Safely store encrypted secrets in your repository",
	Long: `Obscuro encrypts secrets with a password-derived key (Argon2id + AES-256-GCM)
and stores them in .obscuro/secrets.json. Secrets can be injected into templates
by replacing __KEY__ placeholders via stdin/stdout.`,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&password, "password", "p", "", "master password (avoids interactive prompt)")
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// RootCmd exposes the root command for testing.
func RootCmd() *cobra.Command {
	return rootCmd
}

// getPassword returns the password from the flag or prompts interactively.
func getPassword(prompt string) (string, error) {
	if password != "" {
		return password, nil
	}
	fmt.Fprint(os.Stderr, prompt)
	pw, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("reading password: %w", err)
	}
	return string(pw), nil
}

// authenticate loads config, gets password, derives key, and verifies it.
func authenticate() ([]byte, error) {
	if !store.IsInitialized() {
		return nil, fmt.Errorf("not initialized — run 'obscuro init' first")
	}

	cfg, err := store.LoadConfig()
	if err != nil {
		return nil, err
	}

	salt, err := cfg.DecodeSalt()
	if err != nil {
		return nil, fmt.Errorf("decoding salt: %w", err)
	}

	pw, err := getPassword("Enter master password: ")
	if err != nil {
		return nil, err
	}

	key := crypto.DeriveKey(pw, salt)
	if !crypto.VerifyKey(key, cfg.VerificationToken) {
		return nil, fmt.Errorf("incorrect password")
	}

	return key, nil
}
