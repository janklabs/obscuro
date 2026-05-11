package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	injectStrict = false
}

func execCmd(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	rootCmd.SetArgs(args)

	var stdout, stderr bytes.Buffer
	Stdout = &stdout
	rootCmd.SetErr(&stderr)

	err := rootCmd.Execute()
	Stdout = os.Stdout
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

func TestWrongPasswordDoesNotPrintUsage(t *testing.T) {
	setup(t)
	initVault(t)

	_, stderr, err := execCmd(t, "get", "SOMEKEY", "--password", "wrong")
	if err == nil {
		t.Fatal("expected error for wrong password")
	}
	if strings.Contains(stderr, "Usage:") {
		t.Fatalf("stderr should not contain 'Usage:', got: %s", stderr)
	}
	if strings.Contains(stderr, "Global Flags:") {
		t.Fatalf("stderr should not contain 'Global Flags:', got: %s", stderr)
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

func TestGetViaEnvVar(t *testing.T) {
	setup(t)
	initVault(t)

	_, _, err := execCmd(t, "set", "API_KEY", "--password", testPassword, "--value", "env-secret")
	if err != nil {
		t.Fatalf("set failed: %v", err)
	}

	// Use env var instead of --password flag
	os.Setenv("OBSCURO_PASSWORD", testPassword)
	defer os.Unsetenv("OBSCURO_PASSWORD")

	stdout, _, err := execCmd(t, "get", "API_KEY")
	if err != nil {
		t.Fatalf("get via env var failed: %v", err)
	}
	if stdout != "env-secret" {
		t.Fatalf("expected 'env-secret', got %q", stdout)
	}
}

func TestRemove(t *testing.T) {
	setup(t)
	initVault(t)

	_, _, _ = execCmd(t, "set", "API_KEY", "--password", testPassword, "--value", "secret123")
	_, _, _ = execCmd(t, "set", "DB_PASS", "--password", testPassword, "--value", "dbpass456")

	_, stderr, err := execCmd(t, "remove", "API_KEY", "--password", testPassword)
	if err != nil {
		t.Fatalf("remove failed: %v", err)
	}
	if !strings.Contains(stderr, "removed") {
		t.Fatalf("expected 'removed' in stderr, got %q", stderr)
	}

	// Verify it's gone from list
	stdout, _, _ := execCmd(t, "list")
	if strings.Contains(stdout, "API_KEY") {
		t.Fatal("expected API_KEY to be removed from list")
	}
	if !strings.Contains(stdout, "DB_PASS") {
		t.Fatal("expected DB_PASS to remain in list")
	}
}

func TestRemoveNonExistent(t *testing.T) {
	setup(t)
	initVault(t)

	_, _, err := execCmd(t, "remove", "NOPE", "--password", testPassword)
	if err == nil {
		t.Fatal("expected error for removing non-existent key")
	}
}

func TestEnvVarWrongPassword(t *testing.T) {
	setup(t)
	initVault(t)

	_, _, _ = execCmd(t, "set", "API_KEY", "--password", testPassword, "--value", "secret")

	password = ""
	os.Setenv("OBSCURO_PASSWORD", "wrong-password")
	defer os.Unsetenv("OBSCURO_PASSWORD")

	_, _, err := execCmd(t, "get", "API_KEY")
	if err == nil {
		t.Fatal("expected error for wrong env var password")
	}
	if !strings.Contains(err.Error(), "incorrect password") {
		t.Fatalf("expected 'incorrect password' error, got: %v", err)
	}
}

func TestInjectLenientByDefault(t *testing.T) {
	setup(t)
	initVault(t)

	_, _, _ = execCmd(t, "set", "KNOWN", "--password", testPassword, "--value", "known-value")

	input := "a: __KNOWN__\nb: __UNKNOWN__\n"
	rootCmd.SetIn(strings.NewReader(input))

	stdout, stderr, err := execCmd(t, "inject", "--password", testPassword)
	if err != nil {
		t.Fatalf("inject failed: %v", err)
	}
	if !strings.Contains(stdout, "a: known-value") {
		t.Fatalf("expected substituted KNOWN value in stdout, got: %q", stdout)
	}
	if !strings.Contains(stdout, "__UNKNOWN__") {
		t.Fatalf("expected literal __UNKNOWN__ preserved in stdout, got: %q", stdout)
	}
	if !strings.Contains(stderr, "unresolved placeholders: UNKNOWN") {
		t.Fatalf("expected unresolved warning in stderr, got: %q", stderr)
	}
}

func TestInjectStrictFlagFails(t *testing.T) {
	setup(t)
	initVault(t)

	_, _, _ = execCmd(t, "set", "KNOWN", "--password", testPassword, "--value", "known-value")

	input := "a: __KNOWN__\nb: __UNKNOWN__\n"
	rootCmd.SetIn(strings.NewReader(input))

	stdout, stderr, err := execCmd(t, "inject", "--password", testPassword, "--strict")
	if err == nil {
		t.Fatal("expected error in strict mode with unresolved placeholder")
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout in strict mode, got: %q", stdout)
	}
	if !strings.Contains(stderr, "unresolved placeholders: UNKNOWN") {
		t.Fatalf("expected unresolved warning in stderr, got: %q", stderr)
	}
}

func TestInjectStrictEnvFails(t *testing.T) {
	setup(t)
	initVault(t)

	_, _, _ = execCmd(t, "set", "KNOWN", "--password", testPassword, "--value", "known-value")

	input := "a: __KNOWN__\nb: __UNKNOWN__\n"
	rootCmd.SetIn(strings.NewReader(input))

	t.Setenv("OBSCURO_INJECT_STRICT", "1")

	stdout, _, err := execCmd(t, "inject", "--password", testPassword)
	if err == nil {
		t.Fatal("expected error when OBSCURO_INJECT_STRICT=1 with unresolved placeholder")
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout in strict env mode, got: %q", stdout)
	}
}

func TestInjectOnlyDecryptsReferenced(t *testing.T) {
	setup(t)
	initVault(t)

	_, _, _ = execCmd(t, "set", "A", "--password", testPassword, "--value", "alpha")
	_, _, _ = execCmd(t, "set", "B", "--password", testPassword, "--value", "bravo")
	_, _, _ = execCmd(t, "set", "C", "--password", testPassword, "--value", "charlie")

	secrets, err := store.LoadSecrets()
	if err != nil {
		t.Fatalf("load secrets: %v", err)
	}
	secrets["B"] = "!!!"
	if err := store.SaveSecrets(secrets); err != nil {
		t.Fatalf("save secrets: %v", err)
	}

	input := "only: __A__\n"
	rootCmd.SetIn(strings.NewReader(input))

	stdout, _, err := execCmd(t, "inject", "--password", testPassword)
	if err != nil {
		t.Fatalf("inject failed (B should not be decrypted): %v", err)
	}
	if !strings.Contains(stdout, "only: alpha") {
		t.Fatalf("expected substituted A value, got: %q", stdout)
	}
}

func TestAuthClearIdempotent(t *testing.T) {
	setup(t)
	initVault(t)

	_, stderr, err := execCmd(t, "auth", "clear")
	if err != nil {
		t.Fatalf("auth clear should be idempotent, got error: %v", err)
	}
	if !strings.Contains(stderr, "no keychain entry") {
		t.Fatalf("expected stderr to mention 'no keychain entry', got: %q", stderr)
	}
}

func TestAuthStatusShowsSaltAndPath(t *testing.T) {
	if os.Getenv("OBSCURO_TEST_KEYCHAIN") != "1" {
		t.Skip("set OBSCURO_TEST_KEYCHAIN=1 to run keychain-dependent tests")
	}
	setup(t)
	initVault(t)

	if _, _, err := execCmd(t, "auth", "store", "--password", testPassword); err != nil {
		t.Fatalf("auth store failed: %v", err)
	}
	t.Cleanup(func() {
		_, _, _ = execCmd(t, "auth", "clear")
	})

	stdout, _, err := execCmd(t, "auth", "status")
	if err != nil {
		t.Fatalf("auth status failed: %v", err)
	}
	if !strings.Contains(stdout, "Salt fingerprint:") {
		t.Fatalf("expected stdout to contain 'Salt fingerprint:', got: %q", stdout)
	}
	if !strings.Contains(stdout, "Repo:") {
		t.Fatalf("expected stdout to contain 'Repo:', got: %q", stdout)
	}
}

func TestEditTempDirIsPrivate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits not enforced on Windows")
	}
	if _, err := os.Open("/dev/tty"); err != nil {
		t.Skip("no /dev/tty available in this environment")
	}

	setup(t)
	initVault(t)

	if _, _, err := execCmd(t, "set", "MYKEY", "--password", testPassword, "--value", "oldvalue"); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	scriptDir := t.TempDir()
	statOut := filepath.Join(scriptDir, "stat.out")

	var statCmd string
	switch runtime.GOOS {
	case "darwin":
		statCmd = `stat -f '%Lp' "$(dirname "$1")" > ` + statOut + `; stat -f '%Lp' "$1" >> ` + statOut
	default:
		statCmd = `stat -c '%a' "$(dirname "$1")" > ` + statOut + `; stat -c '%a' "$1" >> ` + statOut
	}

	scriptPath := filepath.Join(scriptDir, "fake-editor.sh")
	scriptBody := "#!/bin/sh\n" + statCmd + "\necho newvalue > \"$1\"\n"
	if err := os.WriteFile(scriptPath, []byte(scriptBody), 0o755); err != nil {
		t.Fatalf("writing fake editor: %v", err)
	}

	t.Setenv("EDITOR", scriptPath)

	if _, _, err := execCmd(t, "edit", "MYKEY", "--password", testPassword); err != nil {
		t.Fatalf("edit failed: %v", err)
	}

	data, err := os.ReadFile(statOut)
	if err != nil {
		t.Fatalf("reading stat output: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected 2 stat lines, got: %q", string(data))
	}
	if !strings.Contains(lines[0], "700") {
		t.Fatalf("expected dir perms 700, got: %q", lines[0])
	}
	if !strings.Contains(lines[1], "600") {
		t.Fatalf("expected file perms 600, got: %q", lines[1])
	}
}
