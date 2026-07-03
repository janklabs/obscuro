package cmd

import (
	"strings"
	"testing"

	"github.com/janklabs/obscuro/internal/store"
)

// injectProbes swaps the backendProbesFn seam with a stub that returns the
// supplied keychain and file BackendStatus values verbatim, and returns a
// restore function to be deferred. Tests use this to bypass all real dbus /
// filesystem interaction and lock the detector's row layout in isolation.
func injectProbes(t *testing.T, keychain, file BackendStatus) {
	t.Helper()
	orig := backendProbesFn
	backendProbesFn = func(_ string) (backendProbe, backendProbe) {
		return func() BackendStatus { return keychain },
			func() BackendStatus { return file }
	}
	t.Cleanup(func() { backendProbesFn = orig })
}

// TestDetectBackends_BothAvailable verifies the happy path: both probes
// report Available=true and the detector returns them in the fixed order
// [keychain, file] that the doctor renderer depends on.
func TestDetectBackends_BothAvailable(t *testing.T) {
	injectProbes(t,
		BackendStatus{Kind: BackendKeychain, Name: "OS keychain", Available: true, Reason: "ready"},
		BackendStatus{Kind: BackendFile, Name: "managed file", Available: true, Reason: "ready — writes to /tmp/pw"},
	)

	rows := detectBackends(store.Config{Salt: "test-salt"})
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(rows))
	}
	if !rows[0].Available {
		t.Errorf("rows[0].Available = false, want true")
	}
	if !rows[1].Available {
		t.Errorf("rows[1].Available = false, want true")
	}
	if rows[0].Kind != BackendKeychain {
		t.Errorf("rows[0].Kind = %q, want %q", rows[0].Kind, BackendKeychain)
	}
	if rows[1].Kind != BackendFile {
		t.Errorf("rows[1].Kind = %q, want %q", rows[1].Kind, BackendFile)
	}
}

// TestDetectBackends_KeychainDbusUnreachable verifies the dbus sub-state:
// keychain probe reports the exact remediation string the doctor renders
// when the session bus is missing (headless container, no gnome-keyring).
func TestDetectBackends_KeychainDbusUnreachable(t *testing.T) {
	injectProbes(t,
		BackendStatus{
			Kind:      BackendKeychain,
			Name:      "OS keychain",
			Available: false,
			Reason:    "dbus session bus unreachable — install gnome-keyring or use file backend",
		},
		BackendStatus{Kind: BackendFile, Name: "managed file", Available: true, Reason: "ready"},
	)

	rows := detectBackends(store.Config{Salt: "test-salt"})
	if rows[0].Available {
		t.Errorf("rows[0].Available = true, want false for dbus-unreachable sub-state")
	}
	if !strings.Contains(rows[0].Reason, "dbus") {
		t.Errorf("rows[0].Reason = %q, want it to contain %q", rows[0].Reason, "dbus")
	}
}

// TestDetectBackends_KeychainCollectionLocked verifies the second keychain
// sub-state: Secret Service reachable but the login collection is locked or
// missing (fresh SSH session, first login, no keyring password given).
func TestDetectBackends_KeychainCollectionLocked(t *testing.T) {
	injectProbes(t,
		BackendStatus{
			Kind:      BackendKeychain,
			Name:      "OS keychain",
			Available: false,
			Reason:    "Secret Service reachable but login collection is locked/missing — try 'secret-tool store' first, or use file backend",
		},
		BackendStatus{Kind: BackendFile, Name: "managed file", Available: true, Reason: "ready"},
	)

	rows := detectBackends(store.Config{Salt: "test-salt"})
	if !strings.Contains(rows[0].Reason, "locked") {
		t.Errorf("rows[0].Reason = %q, want it to contain %q", rows[0].Reason, "locked")
	}
}

// TestDetectBackends_KeychainUnsupported verifies the third keychain
// sub-state: the go-keyring library reports the platform is unsupported
// (e.g. bare Alpine without libsecret at all).
func TestDetectBackends_KeychainUnsupported(t *testing.T) {
	injectProbes(t,
		BackendStatus{
			Kind:      BackendKeychain,
			Name:      "OS keychain",
			Available: false,
			Reason:    "unsupported platform",
		},
		BackendStatus{Kind: BackendFile, Name: "managed file", Available: true, Reason: "ready"},
	)

	rows := detectBackends(store.Config{Salt: "test-salt"})
	if rows[0].Reason != "unsupported platform" {
		t.Errorf("rows[0].Reason = %q, want %q", rows[0].Reason, "unsupported platform")
	}
}

// TestDetectBackends_FileUnwritable verifies the file probe failure row:
// the XDG path resolution or write test failed, so the file backend cannot
// be recommended even if the keychain is unhealthy.
func TestDetectBackends_FileUnwritable(t *testing.T) {
	injectProbes(t,
		BackendStatus{Kind: BackendKeychain, Name: "OS keychain", Available: true, Reason: "ready"},
		BackendStatus{Kind: BackendFile, Name: "managed file", Available: false, Reason: "unwritable"},
	)

	rows := detectBackends(store.Config{Salt: "test-salt"})
	if rows[1].Available {
		t.Errorf("rows[1].Available = true, want false when file probe reports unwritable")
	}
}

// TestDetectBackends_FileAvailableAlways_WhenXDGWritable exercises the REAL
// defaultFileProbe against a writable XDG dir. Only the keychain probe is
// stubbed so no dbus contact is attempted; the file probe runs the real
// pwfile.Path + MkdirAll + write-probe path against a t.TempDir() XDG root.
func TestDetectBackends_FileAvailableAlways_WhenXDGWritable(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	orig := backendProbesFn
	backendProbesFn = func(salt string) (backendProbe, backendProbe) {
		_, realFile := defaultBackendProbes(salt)
		stubKeychain := func() BackendStatus {
			return BackendStatus{Kind: BackendKeychain, Name: "OS keychain", Available: true, Reason: "ready"}
		}
		return stubKeychain, realFile
	}
	t.Cleanup(func() { backendProbesFn = orig })

	rows := detectBackends(store.Config{Salt: "test-salt"})
	if !rows[1].Available {
		t.Errorf("rows[1].Available = false, want true with writable XDG_CONFIG_HOME; reason = %q", rows[1].Reason)
	}
}
