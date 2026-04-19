package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/janklabs/obscuro/internal/store"
)

const testPassword = "test-master-pw"

func setup(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	// Initialize a git repo so store.RepoRoot() works
	if out, err := exec.Command("git", "init").CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, out)
	}
	// Reset cached repo root and flags between tests
	store.ResetRoot()
	password = ""
	secretValue = ""
}

func execCmd(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	rootCmd.SetArgs(args)

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)

	err := rootCmd.Execute()
	return stdout.String(), stderr.String(), err
}

func initVault(t *testing.T) {
	t.Helper()
	_, _, err := execCmd(t, "init", "--password", testPassword)
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}
}

func TestInitCreatesDirectory(t *testing.T) {
	setup(t)
	_, stderr, err := execCmd(t, "init", "--password", testPassword)
	if err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr)
	}

	if _, err := os.Stat(".obscuro/config.json"); err != nil {
		t.Fatal("config.json not created")
	}
	if _, err := os.Stat(".obscuro/secrets.json"); err != nil {
		t.Fatal("secrets.json not created")
	}
}

func TestInitAlreadyInitialized(t *testing.T) {
	setup(t)
	initVault(t)

	_, _, err := execCmd(t, "init", "--password", testPassword)
	if err == nil {
		t.Fatal("expected error for double init")
	}
}

func TestSetAndGet(t *testing.T) {
	setup(t)
	initVault(t)

	_, _, err := execCmd(t, "set", "API_KEY", "--password", testPassword, "--value", "my-secret-123")
	if err != nil {
		t.Fatalf("set failed: %v", err)
	}

	stdout, _, err := execCmd(t, "get", "API_KEY", "--password", testPassword)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if stdout != "my-secret-123" {
		t.Fatalf("expected 'my-secret-123', got %q", stdout)
	}
}

func TestGetNonExistent(t *testing.T) {
	setup(t)
	initVault(t)

	_, _, err := execCmd(t, "get", "NOPE", "--password", testPassword)
	if err == nil {
		t.Fatal("expected error for non-existent key")
	}
}

func TestWrongPassword(t *testing.T) {
	setup(t)
	initVault(t)

	_, _, err := execCmd(t, "get", "API_KEY", "--password", "wrong-password")
	if err == nil {
		t.Fatal("expected error for wrong password")
	}
	if !strings.Contains(err.Error(), "incorrect password") {
		t.Fatalf("expected 'incorrect password' error, got: %v", err)
	}
}

func TestList(t *testing.T) {
	setup(t)
	initVault(t)

	_, _, _ = execCmd(t, "set", "ZEBRA", "--password", testPassword, "--value", "z")
	_, _, _ = execCmd(t, "set", "ALPHA", "--password", testPassword, "--value", "a")

	stdout, _, err := execCmd(t, "list")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}

	lines := strings.TrimSpace(stdout)
	if lines != "ALPHA\nZEBRA" {
		t.Fatalf("expected sorted keys, got %q", lines)
	}
}

func TestInject(t *testing.T) {
	setup(t)
	initVault(t)

	_, _, _ = execCmd(t, "set", "API_KEY", "--password", testPassword, "--value", "secret123")
	_, _, _ = execCmd(t, "set", "DB_PASS", "--password", testPassword, "--value", "dbpass456")

	// Provide stdin for inject
	input := "host: example.com\napi_key: __API_KEY__\ndb: __DB_PASS__\n"
	rootCmd.SetIn(strings.NewReader(input))

	stdout, _, err := execCmd(t, "inject", "--password", testPassword)
	if err != nil {
		t.Fatalf("inject failed: %v", err)
	}

	expected := "host: example.com\napi_key: secret123\ndb: dbpass456\n"
	if stdout != expected {
		t.Fatalf("expected:\n%s\ngot:\n%s", expected, stdout)
	}
}

func TestVersion(t *testing.T) {
	setup(t)
	stdout, _, err := execCmd(t, "version")
	if err != nil {
		t.Fatalf("version failed: %v", err)
	}
	if !strings.Contains(stdout, "obscuro") {
		t.Fatalf("expected output to contain 'obscuro', got %q", stdout)
	}
}
