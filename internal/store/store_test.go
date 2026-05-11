package store

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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
	t.Cleanup(ResetRoot)
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

func TestDeleteSecret(t *testing.T) {
	setup(t)
	_ = Init([]byte("0123456789abcdef"), "token")

	secrets := map[string]string{
		"API_KEY": "encrypted1",
		"DB_PASS": "encrypted2",
	}
	if err := SaveSecrets(secrets); err != nil {
		t.Fatal(err)
	}

	if err := DeleteSecret("API_KEY"); err != nil {
		t.Fatalf("DeleteSecret failed: %v", err)
	}

	loaded, err := LoadSecrets()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := loaded["API_KEY"]; ok {
		t.Fatal("expected API_KEY to be deleted")
	}
	if loaded["DB_PASS"] != "encrypted2" {
		t.Fatal("expected DB_PASS to remain")
	}
}

func TestDeleteSecretNotFound(t *testing.T) {
	setup(t)
	_ = Init([]byte("0123456789abcdef"), "token")

	err := DeleteSecret("NONEXISTENT")
	if err == nil {
		t.Fatal("expected error for non-existent key")
	}
}

func TestOutsideGitRepo(t *testing.T) {
	ResetRoot()
	t.Cleanup(ResetRoot)
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	_, err := RepoRoot()
	if err == nil {
		t.Fatal("expected error outside git repo")
	}
	if !strings.Contains(err.Error(), "not inside a git repository") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSubdirResolvesToRepoRoot(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "init").CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, out)
	}
	// Resolve symlinks so comparisons match what `git rev-parse` returns.
	rootResolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(rootResolved, "deep", "nested")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(sub); err != nil {
		t.Fatal(err)
	}
	ResetRoot()
	t.Cleanup(ResetRoot)

	got, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(rootResolved, ".obscuro")
	if got != want {
		t.Fatalf("Dir() = %q, want %q", got, want)
	}
	if !strings.HasSuffix(got, string(filepath.Separator)+".obscuro") {
		t.Fatalf("expected suffix /.obscuro, got %q", got)
	}
}

func TestInitCreatesDir(t *testing.T) {
	setup(t)
	if err := Init([]byte("0123456789abcdef"), "token"); err != nil {
		t.Fatal(err)
	}
	dir, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Fatalf("expected dir, got file")
	}
	if runtime.GOOS != "windows" {
		if perm := info.Mode().Perm(); perm != 0o700 {
			t.Fatalf("expected dir mode 0700, got %o", perm)
		}
	}
}

func TestInitCreatesFilesWithPerms(t *testing.T) {
	setup(t)
	if err := Init([]byte("0123456789abcdef"), "token"); err != nil {
		t.Fatal(err)
	}
	dir, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{ConfigFile, SecretsFile} {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("stat %s: %v", name, err)
		}
		if runtime.GOOS != "windows" {
			if perm := info.Mode().Perm(); perm != 0o600 {
				t.Fatalf("%s: expected mode 0600, got %o", name, perm)
			}
		}
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	setup(t)
	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for missing config file")
	}
}

func TestLoadConfigCorruptJSON(t *testing.T) {
	setup(t)
	if err := Init([]byte("0123456789abcdef"), "token"); err != nil {
		t.Fatal(err)
	}
	dir, _ := Dir()
	if err := os.WriteFile(filepath.Join(dir, ConfigFile), []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "parsing config") {
		t.Fatalf("expected 'parsing config' in error, got: %v", err)
	}
}

// TestLoadSecretsMissingFile verifies LoadSecrets returns an empty map (not an error)
// when secrets.json is absent. This documents the actual behavior.
func TestLoadSecretsMissingFile(t *testing.T) {
	setup(t)
	if err := Init([]byte("0123456789abcdef"), "token"); err != nil {
		t.Fatal(err)
	}
	dir, _ := Dir()
	if err := os.Remove(filepath.Join(dir, SecretsFile)); err != nil {
		t.Fatal(err)
	}
	secrets, err := LoadSecrets()
	if err != nil {
		t.Fatalf("expected no error for missing secrets, got %v", err)
	}
	if len(secrets) != 0 {
		t.Fatalf("expected empty secrets, got %v", secrets)
	}
}

func TestLoadSecretsCorruptJSON(t *testing.T) {
	setup(t)
	if err := Init([]byte("0123456789abcdef"), "token"); err != nil {
		t.Fatal(err)
	}
	dir, _ := Dir()
	if err := os.WriteFile(filepath.Join(dir, SecretsFile), []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadSecrets()
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "parsing secrets") {
		t.Fatalf("expected 'parsing secrets' in error, got: %v", err)
	}
}

func TestSaveSecretsDeterministicWrite(t *testing.T) {
	setup(t)
	if err := Init([]byte("0123456789abcdef"), "token"); err != nil {
		t.Fatal(err)
	}
	if err := SaveSecrets(map[string]string{"B": "2", "A": "1", "C": "3"}); err != nil {
		t.Fatal(err)
	}
	dir, _ := Dir()
	data, err := os.ReadFile(filepath.Join(dir, SecretsFile))
	if err != nil {
		t.Fatal(err)
	}
	idxA := strings.Index(string(data), `"A"`)
	idxB := strings.Index(string(data), `"B"`)
	idxC := strings.Index(string(data), `"C"`)
	if idxA < 0 || idxB < 0 || idxC < 0 {
		t.Fatalf("missing keys in output: %s", data)
	}
	if !(idxA < idxB && idxB < idxC) {
		t.Fatalf("expected keys sorted A<B<C in serialized output: %s", data)
	}
	var got map[string]string
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
}

func TestIsInitializedFalse(t *testing.T) {
	setup(t)
	if IsInitialized() {
		t.Fatal("expected IsInitialized() == false on fresh repo")
	}
}

func TestIsInitializedTrue(t *testing.T) {
	setup(t)
	if err := Init([]byte("0123456789abcdef"), "token"); err != nil {
		t.Fatal(err)
	}
	if !IsInitialized() {
		t.Fatal("expected IsInitialized() == true after Init")
	}
}

func TestOperationsOutsideGitRepo(t *testing.T) {
	ResetRoot()
	t.Cleanup(ResetRoot)
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	if _, err := Dir(); err == nil {
		t.Fatal("Dir() should error outside git repo")
	}
	if IsInitialized() {
		t.Fatal("IsInitialized() should be false outside git repo")
	}
	if err := Init([]byte("salt"), "tok"); err == nil {
		t.Fatal("Init() should error outside git repo")
	}
	if _, err := LoadConfig(); err == nil {
		t.Fatal("LoadConfig() should error outside git repo")
	}
	if _, err := LoadSecrets(); err == nil {
		t.Fatal("LoadSecrets() should error outside git repo")
	}
	if err := SaveSecrets(map[string]string{"K": "V"}); err == nil {
		t.Fatal("SaveSecrets() should error outside git repo")
	}
	if err := DeleteSecret("K"); err == nil {
		t.Fatal("DeleteSecret() should error outside git repo")
	}
}

func TestInitFailsWhenDirIsFile(t *testing.T) {
	setup(t)
	root, err := RepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, DirName), []byte("blocker"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Init([]byte("salt"), "tok"); err == nil {
		t.Fatal("Init() should error when .obscuro is a regular file")
	}
}

func TestInvalidB64Salt(t *testing.T) {
	setup(t)
	if err := Init([]byte("0123456789abcdef"), "x"); err != nil {
		t.Fatal(err)
	}
	dir, _ := Dir()
	bad := []byte(`{"salt":"!!!not-base64!!!","verification_token":"x"}`)
	if err := os.WriteFile(filepath.Join(dir, ConfigFile), bad, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig should succeed for valid JSON, got: %v", err)
	}
	if _, err := cfg.DecodeSalt(); err == nil {
		t.Fatal("expected DecodeSalt to error on invalid base64")
	}
}
