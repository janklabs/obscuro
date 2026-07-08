//go:build unix

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

// withFakeImportChoice replaces runImportChoiceFn with a stub that always
// returns the given choice (no error). Mirrors withFakeBackendChoice at
// cmd_test.go:177. Restored via t.Cleanup.
func withFakeImportChoice(t *testing.T, choice ImportChoice) {
	t.Helper()
	orig := runImportChoiceFn
	runImportChoiceFn = func(_, _ int) (ImportChoice, error) {
		return choice, nil
	}
	t.Cleanup(func() { runImportChoiceFn = orig })
}

// writeEnvFile writes content to a .env file inside t.TempDir() and returns
// the absolute path.
func writeEnvFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	return path
}

// forceTTYStdin swaps os.Stdin to the slave end of a fresh PTY pair so
// isatty.IsTerminal(os.Stdin.Fd()) returns true. Required for tests that
// exercise the TUI branch of `obscuro import`. Restored via t.Cleanup.
// This does NOT run the real bubbletea TUI — tests must also install a
// withFakeImportChoice stub to intercept runImportChoiceFn.
func forceTTYStdin(t *testing.T) {
	t.Helper()
	ptmx, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("open /dev/ptmx: %v", err)
	}
	if err := unix.IoctlSetPointerInt(int(ptmx.Fd()), unix.TIOCSPTLCK, 0); err != nil {
		ptmx.Close()
		t.Fatalf("unlockpt: %v", err)
	}
	n, err := unix.IoctlGetInt(int(ptmx.Fd()), unix.TIOCGPTN)
	if err != nil {
		ptmx.Close()
		t.Fatalf("ptsname: %v", err)
	}
	slave, err := os.OpenFile(fmt.Sprintf("/dev/pts/%d", n), os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		ptmx.Close()
		t.Fatalf("open slave pts: %v", err)
	}
	origStdin := os.Stdin
	os.Stdin = slave
	t.Cleanup(func() {
		os.Stdin = origStdin
		_ = slave.Close()
		_ = ptmx.Close()
	})
}

// resetImportFlags restores the import-command flag default that setup()
// clears. pflag only applies StringVar defaults at registration time, so
// setup()'s `onConflict = ""` would trip the `must be one of skip,
// overwrite, fail` guard on subsequent invocations that omit --on-conflict.
func resetImportFlags() {
	onConflict = "fail"
}

// (a) --------------------------------------------------------------------

func TestImport_NoConflicts_TUIAddsAll(t *testing.T) {
	setup(t)
	initVault(t)
	resetImportFlags()
	forceTTYStdin(t)
	withFakeImportChoice(t, ImportChoiceNewOnly)

	envPath := writeEnvFile(t, "K1=val1\nK2=val2\nK3=val3\n")

	_, stderr, err := execCmd(t, "import", envPath, "--password", testPassword)
	if err != nil {
		t.Fatalf("import failed: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stderr, "Found 3 new secrets and 0 pre-existing secrets") {
		t.Fatalf("stderr missing summary line: %s", stderr)
	}
	if !strings.Contains(stderr, "Import complete: 3 added, 0 overwritten, 0 skipped.") {
		t.Fatalf("stderr missing completion line: %s", stderr)
	}

	stdout, _, err := execCmd(t, "get", "K1", "--password", testPassword)
	if err != nil {
		t.Fatalf("get K1 failed: %v", err)
	}
	if stdout != "val1" {
		t.Fatalf("get K1 = %q, want %q", stdout, "val1")
	}
}

// (b) --------------------------------------------------------------------

func TestImport_ConflictsSkip_TUISkipsExisting(t *testing.T) {
	setup(t)
	initVault(t)
	resetSetFlags()
	if _, _, err := execCmd(t, "set", "EXISTING", "--password", testPassword, "--value", "oldvalue"); err != nil {
		t.Fatalf("seed set failed: %v", err)
	}
	resetSetFlags()
	resetImportFlags()
	forceTTYStdin(t)
	withFakeImportChoice(t, ImportChoiceNewOnly)

	envPath := writeEnvFile(t, "EXISTING=newvalue\nNEWKEY=fresh\n")

	_, stderr, err := execCmd(t, "import", envPath, "--password", testPassword)
	if err != nil {
		t.Fatalf("import failed: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stderr, "Found 1 new secrets and 1 pre-existing secrets") {
		t.Fatalf("stderr missing summary line: %s", stderr)
	}
	if !strings.Contains(stderr, "Import complete: 1 added, 0 overwritten, 1 skipped.") {
		t.Fatalf("stderr missing completion line: %s", stderr)
	}

	stdout, _, err := execCmd(t, "get", "EXISTING", "--password", testPassword)
	if err != nil {
		t.Fatalf("get EXISTING failed: %v", err)
	}
	if stdout != "oldvalue" {
		t.Fatalf("get EXISTING = %q, want %q (should be unchanged)", stdout, "oldvalue")
	}

	stdout, _, err = execCmd(t, "get", "NEWKEY", "--password", testPassword)
	if err != nil {
		t.Fatalf("get NEWKEY failed: %v", err)
	}
	if stdout != "fresh" {
		t.Fatalf("get NEWKEY = %q, want %q", stdout, "fresh")
	}
}

// (c) --------------------------------------------------------------------

func TestImport_ConflictsOverwrite_TUIOverwrites(t *testing.T) {
	setup(t)
	initVault(t)
	resetSetFlags()
	if _, _, err := execCmd(t, "set", "EXISTING", "--password", testPassword, "--value", "oldvalue"); err != nil {
		t.Fatalf("seed set failed: %v", err)
	}
	resetSetFlags()
	resetImportFlags()
	forceTTYStdin(t)
	withFakeImportChoice(t, ImportChoiceOverwrite)

	envPath := writeEnvFile(t, "EXISTING=newvalue\nNEWKEY=fresh\n")

	_, stderr, err := execCmd(t, "import", envPath, "--password", testPassword)
	if err != nil {
		t.Fatalf("import failed: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stderr, "Import complete: 1 added, 1 overwritten, 0 skipped.") {
		t.Fatalf("stderr missing completion line: %s", stderr)
	}

	stdout, _, err := execCmd(t, "get", "EXISTING", "--password", testPassword)
	if err != nil {
		t.Fatalf("get EXISTING failed: %v", err)
	}
	if stdout != "newvalue" {
		t.Fatalf("get EXISTING = %q, want %q (should be overwritten)", stdout, "newvalue")
	}
}

// (d) --------------------------------------------------------------------

func TestImport_Cancel_MakesNoChanges(t *testing.T) {
	setup(t)
	initVault(t)
	resetSetFlags()
	if _, _, err := execCmd(t, "set", "EXISTING", "--password", testPassword, "--value", "oldvalue"); err != nil {
		t.Fatalf("seed set failed: %v", err)
	}
	resetSetFlags()
	resetImportFlags()
	forceTTYStdin(t)
	withFakeImportChoice(t, ImportChoiceCancel)

	envPath := writeEnvFile(t, "EXISTING=newvalue\nNEWKEY=fresh\n")

	_, stderr, err := execCmd(t, "import", envPath, "--password", testPassword)
	if err != nil {
		t.Fatalf("import returned error on cancel: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stderr, "Import cancelled. No changes made.") {
		t.Fatalf("stderr missing cancel line: %s", stderr)
	}

	listOut, _, err := execCmd(t, "list")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if strings.Contains(listOut, "NEWKEY") {
		t.Fatalf("list contains NEWKEY after cancel: %s", listOut)
	}

	stdout, _, err := execCmd(t, "get", "EXISTING", "--password", testPassword)
	if err != nil {
		t.Fatalf("get EXISTING failed: %v", err)
	}
	if stdout != "oldvalue" {
		t.Fatalf("get EXISTING = %q, want %q (should be unchanged)", stdout, "oldvalue")
	}
}

// (e) --------------------------------------------------------------------

func TestImport_OnConflictSkip_NonTTY_table(t *testing.T) {
	cases := []struct {
		name            string
		conflict        string
		envContent      string
		wantErr         bool
		errContains     string
		existingGetsNew bool
	}{
		{
			name:            "skip",
			conflict:        "skip",
			envContent:      "EXISTING=newvalue\nNEWKEY=fresh\n",
			wantErr:         false,
			existingGetsNew: false,
		},
		{
			name:            "overwrite",
			conflict:        "overwrite",
			envContent:      "EXISTING=newvalue\nNEWKEY=fresh\n",
			wantErr:         false,
			existingGetsNew: true,
		},
		{
			name:        "fail-with-conflicts",
			conflict:    "fail",
			envContent:  "EXISTING=newvalue\nNEWKEY=fresh\n",
			wantErr:     true,
			errContains: "existing keys would be overwritten",
		},
		{
			name:       "fail-no-conflicts",
			conflict:   "fail",
			envContent: "ONLYNEW=x\nANOTHER=y\n",
			wantErr:    false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setup(t)
			initVault(t)
			resetSetFlags()
			if _, _, err := execCmd(t, "set", "EXISTING", "--password", testPassword, "--value", "oldvalue"); err != nil {
				t.Fatalf("seed set failed: %v", err)
			}
			resetSetFlags()
			resetImportFlags()

			// Pipe stdin so isatty.IsTerminal returns false and the
			// non-TTY branch (which honors --on-conflict) is taken.
			sr, sw, err := os.Pipe()
			if err != nil {
				t.Fatalf("os.Pipe: %v", err)
			}
			origStdin := os.Stdin
			os.Stdin = sr
			defer func() {
				os.Stdin = origStdin
				_ = sr.Close()
				_ = sw.Close()
			}()

			envPath := writeEnvFile(t, tc.envContent)

			_, stderr, err := execCmd(t, "import", envPath, "--password", testPassword, "--on-conflict", tc.conflict)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil\nstderr: %s", stderr)
				}
				if tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
					t.Fatalf("error = %q, want it to contain %q", err.Error(), tc.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
			}

			// Verify EXISTING key state matches the conflict policy.
			// The "fail-no-conflicts" case has no EXISTING in the file,
			// so EXISTING should still be oldvalue.
			stdout, _, err := execCmd(t, "get", "EXISTING", "--password", testPassword)
			if err != nil {
				t.Fatalf("get EXISTING failed: %v", err)
			}
			want := "oldvalue"
			if tc.existingGetsNew {
				want = "newvalue"
			}
			if stdout != want {
				t.Fatalf("get EXISTING = %q, want %q", stdout, want)
			}
		})
	}
}

// (f) --------------------------------------------------------------------

func TestImport_InvalidKey_ReturnsError(t *testing.T) {
	setup(t)
	initVault(t)
	resetImportFlags()

	envPath := writeEnvFile(t, "foo=bar\n")

	listBefore, _, err := execCmd(t, "list")
	if err != nil {
		t.Fatalf("list (before) failed: %v", err)
	}

	_, _, err = execCmd(t, "import", envPath, "--password", testPassword)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid key names") {
		t.Fatalf("error = %q, want it to contain %q", err.Error(), "invalid key names")
	}
	if !strings.Contains(err.Error(), "foo") {
		t.Fatalf("error = %q, want it to contain %q", err.Error(), "foo")
	}

	listAfter, _, err := execCmd(t, "list")
	if err != nil {
		t.Fatalf("list (after) failed: %v", err)
	}
	if listAfter != listBefore {
		t.Fatalf("list changed after failed import\nbefore: %q\nafter:  %q", listBefore, listAfter)
	}
}

// (g) --------------------------------------------------------------------

func TestImport_EmptyValue_ReturnsError(t *testing.T) {
	setup(t)
	initVault(t)
	resetImportFlags()

	envPath := writeEnvFile(t, "A=\n")

	_, _, err := execCmd(t, "import", envPath, "--password", testPassword)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "empty values not allowed") {
		t.Fatalf("error = %q, want it to contain %q", err.Error(), "empty values not allowed")
	}
	if !strings.Contains(err.Error(), "A") {
		t.Fatalf("error = %q, want it to contain %q", err.Error(), "A")
	}
}

// (h) --------------------------------------------------------------------

func TestImport_MissingFile_ReturnsError(t *testing.T) {
	setup(t)
	initVault(t)
	resetImportFlags()

	_, _, err := execCmd(t, "import", "/nonexistent/path.env", "--password", testPassword)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "parsing .env file") {
		t.Fatalf("error = %q, want it to contain %q", err.Error(), "parsing .env file")
	}
}

// (i) --------------------------------------------------------------------

func TestImport_WrongPassword_FailsBeforeReadingFile(t *testing.T) {
	setup(t)
	initVault(t)
	resetImportFlags()

	envPath := writeEnvFile(t, "GOODKEY=value\n")

	listBefore, _, err := execCmd(t, "list")
	if err != nil {
		t.Fatalf("list (before) failed: %v", err)
	}

	_, _, err = execCmd(t, "import", envPath, "--password", "wrongpassword")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "incorrect password") {
		t.Fatalf("error = %q, want it to contain %q", err.Error(), "incorrect password")
	}

	listAfter, _, err := execCmd(t, "list")
	if err != nil {
		t.Fatalf("list (after) failed: %v", err)
	}
	if listAfter != listBefore {
		t.Fatalf("list changed after failed import\nbefore: %q\nafter:  %q", listBefore, listAfter)
	}
}

// (j) --------------------------------------------------------------------

func TestImport_InvalidOnConflictValue_Rejected(t *testing.T) {
	setup(t)
	initVault(t)
	resetImportFlags()

	envPath := writeEnvFile(t, "K=v\n")

	_, _, err := execCmd(t, "import", envPath, "--password", testPassword, "--on-conflict", "explode")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "must be one of skip, overwrite, fail") {
		t.Fatalf("error = %q, want it to contain %q", err.Error(), "must be one of skip, overwrite, fail")
	}
}

// (k) --------------------------------------------------------------------

func TestImport_MissingArg_ReturnsError(t *testing.T) {
	setup(t)
	initVault(t)
	resetImportFlags()

	_, _, err := execCmd(t, "import", "--password", testPassword)
	if err == nil {
		t.Fatalf("expected cobra arg error, got nil")
	}
}
