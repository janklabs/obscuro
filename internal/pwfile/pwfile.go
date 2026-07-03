// Package pwfile implements a managed password-file store used as an
// alternative to the OS keychain. Files are written atomically with
// mode 0600 under an XDG-resolved base directory.
package pwfile

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

// ErrNotFound is returned by Read when no password file exists for the
// given salt.
var ErrNotFound = errors.New("password file not found")

// Path returns the absolute path to the password file for the given
// base64-encoded salt.
func Path(saltB64 string) (string, error) {
	return filepath.Join(baseDir(), "vaults", hash(saltB64)+".pw"), nil
}

// Write persists the password for the given salt to disk atomically.
// The parent directory is created with mode 0700 and the file with
// mode 0600.
func Write(saltB64, password string) error {
	path, err := Path(saltB64)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write([]byte(password)); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// Read returns the password stored for the given salt. It returns
// ErrNotFound if no password file exists.
func Read(saltB64 string) (string, error) {
	path, err := Path(saltB64)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", ErrNotFound
		}
		return "", err
	}
	return string(data), nil
}

// Delete removes the password file for the given salt. A missing file
// is treated as success.
func Delete(saltB64 string) error {
	path, err := Path(saltB64)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	return nil
}

// Exists reports whether a password file exists for the given salt.
func Exists(saltB64 string) bool {
	path, err := Path(saltB64)
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// baseDir returns the obscuro configuration base directory. On Windows
// it uses os.UserConfigDir (%AppData%); on all other platforms it
// resolves XDG_CONFIG_HOME manually, falling back to ~/.config. macOS
// deliberately uses ~/.config rather than ~/Library/Application Support
// for XDG consistency across Unix-like systems.
func baseDir() string {
	if runtime.GOOS == "windows" {
		dir, _ := os.UserConfigDir()
		return filepath.Join(dir, "obscuro")
	}
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		home, _ := os.UserHomeDir()
		xdg = filepath.Join(home, ".config")
	}
	return filepath.Join(xdg, "obscuro")
}

// hash returns the first 16 hex characters of the SHA-256 digest of the
// salt. Used to derive per-vault filenames without exposing the salt.
func hash(saltB64 string) string {
	sum := sha256.Sum256([]byte(saltB64))
	return hex.EncodeToString(sum[:])[:16]
}
