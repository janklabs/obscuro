package store

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

const (
	DirName     = ".obscuro"
	ConfigFile  = "config.json"
	SecretsFile = "secrets.json"
)

var (
	repoRoot     string
	repoRootOnce sync.Once
	repoRootErr  error
)

// RepoRoot returns the git repository root directory.
// It caches the result after the first call.
func RepoRoot() (string, error) {
	repoRootOnce.Do(func() {
		out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
		if err != nil {
			repoRootErr = fmt.Errorf("not inside a git repository")
			return
		}
		repoRoot = strings.TrimSpace(string(out))
	})
	return repoRoot, repoRootErr
}

// ResetRoot clears the cached repo root. Used in tests.
func ResetRoot() {
	repoRootOnce = sync.Once{}
	repoRoot = ""
	repoRootErr = nil
}

// Config holds the Argon2id salt and verification token.
type Config struct {
	Salt              string `json:"salt"`               // base64-encoded
	VerificationToken string `json:"verification_token"` // base64(nonce||ciphertext)
}

// Dir returns the absolute path to the .obscuro directory at the repo root.
func Dir() (string, error) {
	root, err := RepoRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, DirName), nil
}

func configPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ConfigFile), nil
}

func secretsPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, SecretsFile), nil
}

// IsInitialized returns true if the .obscuro directory and config exist.
func IsInitialized() bool {
	p, err := configPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}

// Init creates the .obscuro directory, config, and empty secrets file.
func Init(salt []byte, verificationToken string) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	cp, err := configPath()
	if err != nil {
		return err
	}
	cfg := Config{
		Salt:              base64.StdEncoding.EncodeToString(salt),
		VerificationToken: verificationToken,
	}
	if err := writeJSON(cp, cfg); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	sp, err := secretsPath()
	if err != nil {
		return err
	}
	if err := writeJSON(sp, map[string]string{}); err != nil {
		return fmt.Errorf("writing secrets: %w", err)
	}
	return nil
}

// LoadConfig reads and parses the config file.
func LoadConfig() (*Config, error) {
	p, err := configPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}

// DecodeSalt returns the decoded salt from config.
func (c *Config) DecodeSalt() ([]byte, error) {
	return base64.StdEncoding.DecodeString(c.Salt)
}

// LoadSecrets reads the secrets file into a map.
func LoadSecrets() (map[string]string, error) {
	p, err := secretsPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("reading secrets: %w", err)
	}
	var secrets map[string]string
	if err := json.Unmarshal(data, &secrets); err != nil {
		return nil, fmt.Errorf("parsing secrets: %w", err)
	}
	return secrets, nil
}

// SaveSecrets writes the secrets map to disk.
func SaveSecrets(secrets map[string]string) error {
	p, err := secretsPath()
	if err != nil {
		return err
	}
	return writeJSON(p, secrets)
}

// ListKeys returns sorted secret key names.
func ListKeys(secrets map[string]string) []string {
	keys := make([]string, 0, len(secrets))
	for k := range secrets {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func writeJSON(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
