package store

import (
	"os"
	"os/exec"
	"testing"
)

func setup(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	// Initialize a git repo so RepoRoot() works
	if out, err := exec.Command("git", "init").CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, out)
	}
	// Reset cached repo root from previous tests
	ResetRoot()
}

func TestIsInitializedFalseBeforeInit(t *testing.T) {
	setup(t)
	if IsInitialized() {
		t.Fatal("expected not initialized")
	}
}

func TestInitCreatesFiles(t *testing.T) {
	setup(t)

	salt := []byte("0123456789abcdef")
	err := Init(salt, "fake-token")
	if err != nil {
		t.Fatal(err)
	}

	if !IsInitialized() {
		t.Fatal("expected initialized after Init")
	}

	// Check config
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.VerificationToken != "fake-token" {
		t.Fatalf("unexpected token: %s", cfg.VerificationToken)
	}

	// Check secrets
	secrets, err := LoadSecrets()
	if err != nil {
		t.Fatal(err)
	}
	if len(secrets) != 0 {
		t.Fatal("expected empty secrets")
	}
}

func TestSaveAndLoadSecrets(t *testing.T) {
	setup(t)
	_ = Init([]byte("0123456789abcdef"), "token")

	secrets := map[string]string{
		"API_KEY": "encrypted1",
		"DB_PASS": "encrypted2",
	}
	if err := SaveSecrets(secrets); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadSecrets()
	if err != nil {
		t.Fatal(err)
	}
	if loaded["API_KEY"] != "encrypted1" || loaded["DB_PASS"] != "encrypted2" {
		t.Fatalf("unexpected secrets: %v", loaded)
	}
}

func TestListKeysSorted(t *testing.T) {
	secrets := map[string]string{
		"ZEBRA":  "z",
		"ALPHA":  "a",
		"MIDDLE": "m",
	}
	keys := ListKeys(secrets)
	if len(keys) != 3 || keys[0] != "ALPHA" || keys[1] != "MIDDLE" || keys[2] != "ZEBRA" {
		t.Fatalf("expected sorted keys, got %v", keys)
	}
}
