package cmd

// These tests cover the two thin surfaces of backend_tui.go that DO NOT
// require a real terminal: the runBackendSelectorFn seam (dependency
// injection point mirroring promptPasswordFn) and the non-TTY refusal
// branch of runBackendSelector. The bubbletea list itself is intentionally
// NOT tested here — driving it would need a PTY, and its behavior is a
// property of the upstream library, not our code.

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

// TestRunBackendSelectorFn_ReturnsScriptedChoice verifies the test seam:
// swapping runBackendSelectorFn with a deterministic stub lets callers
// simulate a successful selection without spinning a bubbletea program.
// This is the pattern every command that invokes the selector uses in its
// own tests.
func TestRunBackendSelectorFn_ReturnsScriptedChoice(t *testing.T) {
	orig := runBackendSelectorFn
	runBackendSelectorFn = func(_ []BackendStatus, _ bool) (backendChoice, error) {
		return backendChoice{Kind: BackendFile}, nil
	}
	t.Cleanup(func() { runBackendSelectorFn = orig })

	choice, err := runBackendSelectorFn(nil, false)
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if choice.Kind != BackendFile {
		t.Errorf("choice.Kind = %q, want %q", choice.Kind, BackendFile)
	}
}

// TestRunBackendSelectorFn_ReturnsCancelled verifies the seam propagates
// ErrCancelled unchanged. Callers rely on errors.Is to distinguish user
// cancellation from other failure modes (e.g. the non-TTY refusal), so the
// sentinel identity must survive the indirection.
func TestRunBackendSelectorFn_ReturnsCancelled(t *testing.T) {
	orig := runBackendSelectorFn
	runBackendSelectorFn = func(_ []BackendStatus, _ bool) (backendChoice, error) {
		return backendChoice{}, ErrCancelled
	}
	t.Cleanup(func() { runBackendSelectorFn = orig })

	_, err := runBackendSelectorFn(nil, false)
	if !errors.Is(err, ErrCancelled) {
		t.Errorf("errors.Is(err, ErrCancelled) = false; err = %v", err)
	}
}

// TestRunBackendSelector_NonInteractive drives the REAL runBackendSelector
// with a piped stdin so isatty.IsTerminal(os.Stdin.Fd()) returns false.
// The pipe is the mechanism — any non-tty *os.File would do, but a pipe is
// the cheapest and most portable choice. Stderr is redirected to a second
// pipe so we can assert the exact remediation hint the user sees when the
// TUI cannot run (CI, piped stdin, headless container).
func TestRunBackendSelector_NonInteractive(t *testing.T) {
	sr, sw, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe stdin: %v", err)
	}
	origStdin := os.Stdin
	os.Stdin = sr
	defer func() {
		os.Stdin = origStdin
		sr.Close()
		sw.Close()
	}()

	er, ew, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe stderr: %v", err)
	}
	origStderr := os.Stderr
	os.Stderr = ew
	defer func() { os.Stderr = origStderr }()

	statuses := []BackendStatus{
		{Kind: BackendKeychain, Name: "OS keychain", Available: true, Reason: "ready"},
		{Kind: BackendFile, Name: "managed file", Available: true, Reason: "ready"},
	}

	_, callErr := runBackendSelector(statuses, false)

	// Close the write end so io.Copy on the read end terminates instead of
	// blocking waiting for more bytes. The read end is closed by the
	// deferred cleanup after the assertions run.
	if err := ew.Close(); err != nil {
		t.Fatalf("close stderr write end: %v", err)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, er); err != nil {
		t.Fatalf("copy stderr: %v", err)
	}
	if err := er.Close(); err != nil {
		t.Fatalf("close stderr read end: %v", err)
	}

	if !errors.Is(callErr, ErrNonInteractive) {
		t.Errorf("errors.Is(err, ErrNonInteractive) = false; err = %v", callErr)
	}

	got := buf.String()
	if !strings.Contains(got, "--backend=keychain") {
		t.Errorf("stderr = %q, want it to contain %q", got, "--backend=keychain")
	}
	if !strings.Contains(got, "--backend=file") {
		t.Errorf("stderr = %q, want it to contain %q", got, "--backend=file")
	}
}
