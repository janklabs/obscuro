package pwfile

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// setXDG points the pwfile base directory at a fresh per-test temp dir so
// no test touches the real ~/.config. On Windows os.UserConfigDir reads
// %AppData%, so that variable is set instead.
func setXDG(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", dir)
	} else {
		t.Setenv("XDG_CONFIG_HOME", dir)
	}
	return dir
}

// TestWriteRead_Roundtrip verifies a value written under a salt is read back verbatim.
func TestWriteRead_Roundtrip(t *testing.T) {
	setXDG(t)

	if err := Write("saltA", "pw"); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	got, err := Read("saltA")
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if got != "pw" {
		t.Errorf("Read(%q) = %q, want %q", "saltA", got, "pw")
	}
}

// TestFileMode0600 verifies the password file is written with 0600 permissions on unix.
func TestFileMode0600(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file mode check skipped on Windows")
	}
	setXDG(t)

	if err := Write("saltA", "pw"); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	path, err := Path("saltA")
	if err != nil {
		t.Fatalf("Path failed: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q) failed: %v", path, err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("file mode = %o, want 0600", perm)
	}
}

// TestParentDirCreated verifies Write creates <XDG>/obscuro/vaults with mode 0700 on unix.
func TestParentDirCreated(t *testing.T) {
	xdg := setXDG(t)

	if err := Write("saltA", "pw"); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	vaultsDir := filepath.Join(xdg, "obscuro", "vaults")
	info, err := os.Stat(vaultsDir)
	if err != nil {
		t.Fatalf("Stat(%q) failed: %v", vaultsDir, err)
	}
	if !info.IsDir() {
		t.Fatalf("%q is not a directory", vaultsDir)
	}
	if runtime.GOOS == "windows" {
		return
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Errorf("dir mode = %o, want 0700", perm)
	}
}

// TestReadMissingReturnsErrNotFound verifies Read on an unknown salt returns ErrNotFound.
func TestReadMissingReturnsErrNotFound(t *testing.T) {
	setXDG(t)

	_, err := Read("neverwritten")
	if err == nil {
		t.Fatal("Read on missing file returned nil error")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("errors.Is(err, ErrNotFound) = false; err = %v", err)
	}
}

// TestDeleteIdempotent verifies Delete on a missing file returns nil, and repeated
// Delete calls after a Write both succeed.
func TestDeleteIdempotent(t *testing.T) {
	setXDG(t)

	if err := Delete("neverwritten"); err != nil {
		t.Errorf("Delete on missing file: got %v, want nil", err)
	}

	if err := Write("saltA", "pw"); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if err := Delete("saltA"); err != nil {
		t.Errorf("first Delete after Write: got %v, want nil", err)
	}
	if err := Delete("saltA"); err != nil {
		t.Errorf("second Delete after Write: got %v, want nil", err)
	}
}

// TestExistsAfterWrite verifies Exists tracks Write and Delete.
func TestExistsAfterWrite(t *testing.T) {
	setXDG(t)

	if err := Write("saltA", "pw"); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if !Exists("saltA") {
		t.Error("Exists after Write = false, want true")
	}

	if err := Delete("saltA"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if Exists("saltA") {
		t.Error("Exists after Delete = true, want false")
	}
}

// TestWriteOverwrites verifies a second Write for the same salt replaces the first.
func TestWriteOverwrites(t *testing.T) {
	setXDG(t)

	if err := Write("saltA", "pw1"); err != nil {
		t.Fatalf("first Write failed: %v", err)
	}
	if err := Write("saltA", "pw2"); err != nil {
		t.Fatalf("second Write failed: %v", err)
	}

	got, err := Read("saltA")
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if got != "pw2" {
		t.Errorf("Read(%q) = %q, want %q", "saltA", got, "pw2")
	}
}

// TestDifferentSaltsIsolated verifies entries under different salts do not collide.
func TestDifferentSaltsIsolated(t *testing.T) {
	setXDG(t)

	if err := Write("saltA", "pw1"); err != nil {
		t.Fatalf("Write saltA failed: %v", err)
	}
	if err := Write("saltB", "pw2"); err != nil {
		t.Fatalf("Write saltB failed: %v", err)
	}

	gotA, err := Read("saltA")
	if err != nil {
		t.Fatalf("Read saltA failed: %v", err)
	}
	if gotA != "pw1" {
		t.Errorf("Read(saltA) = %q, want %q", gotA, "pw1")
	}

	gotB, err := Read("saltB")
	if err != nil {
		t.Fatalf("Read saltB failed: %v", err)
	}
	if gotB != "pw2" {
		t.Errorf("Read(saltB) = %q, want %q", gotB, "pw2")
	}
}
