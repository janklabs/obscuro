package cmd

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/janklabs/obscuro/internal/crypto"
	"github.com/janklabs/obscuro/internal/keychain"
	"github.com/janklabs/obscuro/internal/store"
	"github.com/janklabs/obscuro/internal/version"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
	"golang.org/x/term"
)

var password string

// Stdout is the writer for command output (payload data).
// Tests can override this to capture output.
var Stdout io.Writer = os.Stdout

// updateResult carries the result of a background update check.
type updateResult struct {
	latest string
	err    error
}

var updateCh chan updateResult

var rootCmd = &cobra.Command{
	Use:   "obscuro",
	Short: "Safely store encrypted secrets in your repository",
	Long: `Obscuro encrypts secrets with a password-derived key (Argon2id + AES-256-GCM)
and stores them in .obscuro/secrets.json. Secrets can be injected into templates
by replacing __KEY__ placeholders via stdin/stdout.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Skip for commands that already handle versioning.
		name := cmd.Name()
		if name == "upgrade" || name == "version" {
			return
		}

		// Skip if opted out or running a dev build.
		if os.Getenv("OBSCURO_NO_UPDATE_CHECK") == "1" {
			return
		}
		if version.Version == "dev" {
			return
		}

		updateCh = make(chan updateResult, 1)
		go func() {
			latest, err := fetchLatestTag()
			updateCh <- updateResult{latest: latest, err: err}
		}()
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if updateCh == nil {
			return
		}

		select {
		case res := <-updateCh:
			if res.err != nil || res.latest == "" {
				return
			}
			if semver.Compare(version.Version, res.latest) < 0 {
				fmt.Fprintf(os.Stderr, "\nA new version of obscuro is available: %s (current: %s). Run 'obscuro upgrade' to update.\n", res.latest, version.Version)
			}
		case <-time.After(2 * time.Second):
			// Don't block if the check is slow.
		}
	},
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

// getPassword resolves the master password using the following priority:
//  1. --password / -p flag
//  2. OS keychain (keyed by vault salt)
//  3. OBSCURO_PASSWORD environment variable
//  4. Interactive /dev/tty prompt
func getPassword(prompt string, salt string) (string, error) {
	// 1. Flag
	if password != "" {
		return password, nil
	}

	// 2. OS keychain
	if salt != "" {
		if pw, err := keychain.Get(salt); err == nil && pw != "" {
			return pw, nil
		}
	}

	// 3. Environment variable
	if pw := os.Getenv("OBSCURO_PASSWORD"); pw != "" {
		return pw, nil
	}

	// 4. Interactive prompt
	tty, err := openTTY()
	if err != nil {
		return "", fmt.Errorf("cannot open terminal for password prompt: %w", err)
	}
	defer tty.Close()

	fmt.Fprint(tty, prompt)
	pw, err := term.ReadPassword(int(tty.Fd()))
	fmt.Fprintln(tty)
	if err != nil {
		return "", fmt.Errorf("reading password: %w", err)
	}
	return string(pw), nil
}

// openTTY opens the terminal device directly, bypassing stdin.
func openTTY() (*os.File, error) {
	if runtime.GOOS == "windows" {
		return os.Open("CONIN$")
	}
	return os.Open("/dev/tty")
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

	pw, err := getPassword("Enter master password: ", cfg.Salt)
	if err != nil {
		return nil, err
	}

	key := crypto.DeriveKey(pw, salt)
	if !crypto.VerifyKey(key, cfg.VerificationToken) {
		return nil, fmt.Errorf("incorrect password")
	}

	return key, nil
}
