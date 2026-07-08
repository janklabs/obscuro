package cmd

// These tests cover the two thin surfaces of import_tui.go that DO NOT
// require a real terminal: the runImportChoiceFn seam (dependency
// injection point mirroring runBackendSelectorFn / promptPasswordFn) and
// the non-TTY refusal branch of runImportChoice. The bubbletea list
// itself is intentionally NOT tested here — driving it would need a PTY,
// and its behavior is a property of the upstream library, not our code.

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// TestRunImportChoiceFn_ReturnsScriptedChoice verifies the test seam:
// swapping runImportChoiceFn with a deterministic stub lets callers
// simulate a successful selection without spinning a bubbletea program.
// This is the pattern the import command uses in its own tests. Each of
// the three ImportChoice values is exercised so a caller that
// switch-cases on the return value cannot regress in silence.
func TestRunImportChoiceFn_ReturnsScriptedChoice(t *testing.T) {
	cases := []ImportChoice{
		ImportChoiceNewOnly,
		ImportChoiceOverwrite,
		ImportChoiceCancel,
	}
	for _, want := range cases {
		want := want
		t.Run(string(want), func(t *testing.T) {
			orig := runImportChoiceFn
			runImportChoiceFn = func(_, _ int) (ImportChoice, error) {
				return want, nil
			}
			t.Cleanup(func() { runImportChoiceFn = orig })

			got, err := runImportChoiceFn(0, 0)
			if err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
			if got != want {
				t.Errorf("choice = %q, want %q", got, want)
			}
		})
	}
}

// TestRunImportChoiceFn_ReturnsCancelled verifies the seam propagates
// ErrCancelled unchanged. Callers rely on errors.Is to distinguish user
// cancellation from other failure modes (e.g. the non-TTY refusal), so
// the sentinel identity must survive the indirection.
func TestRunImportChoiceFn_ReturnsCancelled(t *testing.T) {
	orig := runImportChoiceFn
	runImportChoiceFn = func(_, _ int) (ImportChoice, error) {
		return ImportChoiceCancel, ErrCancelled
	}
	t.Cleanup(func() { runImportChoiceFn = orig })

	_, err := runImportChoiceFn(0, 0)
	if !errors.Is(err, ErrCancelled) {
		t.Errorf("errors.Is(err, ErrCancelled) = false; err = %v", err)
	}
}

// TestRunImportChoice_NonInteractive drives the REAL runImportChoice
// with a piped stdin so isatty.IsTerminal(os.Stdin.Fd()) returns false.
// The pipe is the mechanism — any non-tty *os.File would do, but a pipe
// is the cheapest and most portable choice. Stderr is redirected to a
// second pipe so we can assert the exact remediation hint the user sees
// when the TUI cannot run (CI, piped stdin, headless container).
func TestRunImportChoice_NonInteractive(t *testing.T) {
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

	_, callErr := runImportChoice(3, 2)

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
	if !strings.Contains(got, "--on-conflict") {
		t.Errorf("stderr = %q, want it to contain %q", got, "--on-conflict")
	}
	if !strings.Contains(got, "terminal") && !strings.Contains(got, "TTY") {
		t.Errorf("stderr = %q, want it to contain %q or %q", got, "terminal", "TTY")
	}
}

// TestImportModel_WindowResize_UsesCompactHeight locks the compact
// rendering behaviour. Regression guard: an earlier version used the
// full terminal viewport (msg.Height - v) instead of compactHeight(),
// which made the prompt paint ~30 blank rows below the footer on a
// standard 40-row terminal. This test drives the Update handler
// directly (no bubbletea program needed) and asserts the list height
// stays at the compact bound for both 2- and 3-item variants.
func TestImportModel_WindowResize_UsesCompactHeight(t *testing.T) {
	const bigTerminalHeight = 40
	cases := []struct {
		name   string
		nItems int
	}{
		{"2-item prompt (no conflicts)", 2},
		{"3-item prompt (with conflicts)", 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			items := make([]list.Item, tc.nItems)
			for i := range items {
				items[i] = importItem{title: "opt", choice: ImportChoiceCancel}
			}
			delegate := list.NewDefaultDelegate()
			delegate.ShowDescription = false
			m := importModel{nItems: tc.nItems}
			m.list = list.New(items, delegate, 60, compactHeight(tc.nItems, false, nil))

			updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: bigTerminalHeight})
			fm, ok := updated.(importModel)
			if !ok {
				t.Fatalf("unexpected model type %T", updated)
			}

			want := compactHeight(tc.nItems, false, nil)
			if fm.list.Height() != want {
				t.Errorf("list height = %d, want compactHeight = %d (regression: prompt is consuming full terminal viewport)", fm.list.Height(), want)
			}
			if fm.list.Height() >= bigTerminalHeight {
				t.Errorf("list height %d >= terminal height %d — prompt is not compact", fm.list.Height(), bigTerminalHeight)
			}
		})
	}
}
