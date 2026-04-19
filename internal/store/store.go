package store

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const (
	DirName     = ".obscuro"
	ConfigFile  = "config.json"
	SecretsFile = "secrets.json"
)

// Config holds the Argon2id salt and verification token.
type Config struct {
	Salt              string `json:"salt"`               // base64-encoded
	VerificationToken string `json:"verification_token"` // base64(nonce||ciphertext)
}

// Dir returns the .obscuro directory path relative to the current working directory.
func Dir() string {
	return DirName
}

func configPath() string {
	return filepath.Join(DirName, ConfigFile)
}

func secretsPath() string {
	return filepath.Join(DirName, SecretsFile)
}

// IsInitialized returns true if the .obscuro directory and config exist.
func IsInitialized() bool {
	_, err := os.Stat(configPath())
	return err == nil
}

// Init creates the .obscuro directory, config, and empty secrets file.
func Init(salt []byte, verificationToken string) error {
	if err := os.MkdirAll(DirName, 0o700); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}
	cfg := Config{
		Salt:              base64.StdEncoding.EncodeToString(salt),
		VerificationToken: verificationToken,
	}
	if err := writeJSON(configPath(), cfg); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	if err := writeJSON(secretsPath(), map[string]string{}); err != nil {
		return fmt.Errorf("writing secrets: %w", err)
	}
	return nil
}

// LoadConfig reads and parses the config file.
func LoadConfig() (*Config, error) {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}

// Salt returns the decoded salt from config.
func (c *Config) DecodeSalt() ([]byte, error) {
	return base64.StdEncoding.DecodeString(c.Salt)
}

// LoadSecrets reads the secrets file into a map.
func LoadSecrets() (map[string]string, error) {
	data, err := os.ReadFile(secretsPath())
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
	return writeJSON(secretsPath(), secrets)
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
