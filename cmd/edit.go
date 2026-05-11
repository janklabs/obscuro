package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/janklabs/obscuro/internal/crypto"
	"github.com/janklabs/obscuro/internal/store"
	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit KEY",
	Short: "Edit a secret in your default editor",
	Long: "Edits a secret in your default editor ($EDITOR or vi).\n\n" +
		"The decrypted value is written to a temporary file inside a private 0700 directory with mode 0600. " +
		"After the editor exits, the file is overwritten with zeros and removed.\n\n" +
		"This is a defense-in-depth measure, NOT a forensic guarantee: editor swap/backup files may write copies elsewhere, " +
		"and copy-on-write filesystems (APFS, btrfs, ZFS) may retain old blocks despite the overwrite.",
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

		dir, err := os.MkdirTemp("", "obscuro-edit-")
		if err != nil {
			return fmt.Errorf("creating temp dir: %w", err)
		}
		if err := os.Chmod(dir, 0o700); err != nil {
			os.RemoveAll(dir)
			return fmt.Errorf("setting temp dir permissions: %w", err)
		}
		tmpPath := filepath.Join(dir, "secret")
		tmp, err := os.OpenFile(tmpPath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			os.RemoveAll(dir)
			return fmt.Errorf("creating temp file: %w", err)
		}

		var edited []byte
		// Defers run LIFO: register RemoveAll first so it runs LAST,
		// then register zeroization so it runs FIRST.
		defer os.RemoveAll(dir)
		defer func() {
			f, err := os.OpenFile(tmpPath, os.O_WRONLY, 0)
			if err != nil {
				return // best-effort
			}
			size := len(plaintext)
			if len(edited) > size {
				size = len(edited)
			}
			_, _ = f.Write(bytes.Repeat([]byte{0}, size))
			_ = f.Sync()
			_ = f.Close()
		}()

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
		edited, err = os.ReadFile(tmpPath)
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
