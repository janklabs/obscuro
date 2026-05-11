package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/janklabs/obscuro/internal/crypto"
	"github.com/janklabs/obscuro/internal/store"
	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:          "edit KEY",
	Short:        "Edit a secret in your default editor",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		key, err := authenticate()
		if err != nil {
			return err
		}

		secrets, err := store.LoadSecrets()
		if err != nil {
			return err
		}

		encrypted, ok := secrets[name]
		if !ok {
			return fmt.Errorf("secret '%s' not found", name)
		}

		plaintext, err := crypto.Decrypt(key, encrypted)
		if err != nil {
			return fmt.Errorf("decrypting secret: %w", err)
		}

		// Write plaintext to a temporary file with restrictive permissions.
		tmp, err := os.CreateTemp("", "obscuro-edit-*")
		if err != nil {
			return fmt.Errorf("creating temp file: %w", err)
		}
		tmpPath := tmp.Name()
		defer os.Remove(tmpPath)

		if err := os.Chmod(tmpPath, 0600); err != nil {
			tmp.Close()
			return fmt.Errorf("setting temp file permissions: %w", err)
		}
		if _, err := tmp.Write(plaintext); err != nil {
			tmp.Close()
			return fmt.Errorf("writing temp file: %w", err)
		}
		if err := tmp.Close(); err != nil {
			return fmt.Errorf("closing temp file: %w", err)
		}

		// Resolve editor.
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}

		// Open the editor, attaching to the TTY for interactive use.
		tty, err := openTTY()
		if err != nil {
			return fmt.Errorf("cannot open terminal: %w", err)
		}
		defer tty.Close()

		c := exec.Command(editor, tmpPath)
		c.Stdin = tty
		c.Stdout = tty
		c.Stderr = tty
		if err := c.Run(); err != nil {
			return fmt.Errorf("editor exited with error, changes discarded: %w", err)
		}

		// Read back the edited value.
		edited, err := os.ReadFile(tmpPath)
		if err != nil {
			return fmt.Errorf("reading temp file: %w", err)
		}

		if string(edited) == string(plaintext) {
			fmt.Fprintln(os.Stderr, "No changes.")
			return nil
		}

		if len(edited) == 0 {
			return fmt.Errorf("secret value cannot be empty")
		}

		newEncrypted, err := crypto.Encrypt(key, edited)
		if err != nil {
			return err
		}

		secrets[name] = newEncrypted
		if err := store.SaveSecrets(secrets); err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "Secret '%s' updated.\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(editCmd)
}
