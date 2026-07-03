package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/janklabs/obscuro/internal/keychain"
	"github.com/janklabs/obscuro/internal/pwfile"
	"github.com/janklabs/obscuro/internal/store"
	keyring "github.com/zalando/go-keyring"
)

// BackendKind identifies a password-backend row surfaced by the doctor/status
// view. It is intentionally a stringly-typed enum so JSON output stays stable.
type BackendKind string

const (
	BackendKeychain BackendKind = "keychain"
	BackendFile     BackendKind = "file"
	BackendNone     BackendKind = "none"
)

// BackendStatus is a single detected-backend row. Kind + Name describe *what*
// was probed; Available + Reason summarise the outcome for one-line display;
// Verbose collects every diagnostic the user should see under `--verbose`.
type BackendStatus struct {
	Kind      BackendKind
	Name      string
	Reason    string
	Available bool
	Verbose   []string
}

// backendProbe returns the current status for a single backend. The signature
// is closed over configuration (e.g. the vault salt) at construction time so
// the caller can invoke it with zero arguments.
type backendProbe func() BackendStatus

// backendProbesFn is the seam swapped in tests. It returns the keychain probe
// first and the file probe second. The salt is threaded through so the file
// probe closure touches the exact directory that would host this vault's
// pwfile — see cmd/os_detect.go::osDetectFn for the same pattern.
var backendProbesFn func(salt string) (keychainProbe, fileProbe backendProbe) = defaultBackendProbes

// detectBackends runs each configured backend probe and returns the results
// in the fixed order [keychain, file]. Callers rely on this ordering for
// stable table rendering.
func detectBackends(cfg store.Config) []BackendStatus {
	kp, fp := backendProbesFn(cfg.Salt)
	return []BackendStatus{kp(), fp()}
}

// defaultBackendProbes returns the production keychain and file probes. The
// salt is bound to the file probe closure so it resolves to the vault's
// pwfile path even if cfg.Salt is later mutated.
func defaultBackendProbes(salt string) (keychainProbe, fileProbe backendProbe) {
	return defaultKeychainProbe(), defaultFileProbe(salt)
}

// defaultKeychainProbe returns a closure that probes the OS keychain by
// attempting a trial Set with the exported ServiceName and a sentinel user.
// It classifies failure into exactly three sub-states — unsupported platform,
// dbus unreachable, collection locked/missing — plus a generic fallback so
// `obscuro doctor` can render an actionable next step per host.
func defaultKeychainProbe() backendProbe {
	return func() BackendStatus {
		status := BackendStatus{
			Kind: BackendKeychain,
			Name: "OS keychain",
		}
		err := keyring.Set(keychain.ServiceName, "__obscuro_probe__", "")
		status.Verbose = append(status.Verbose,
			"DBUS_SESSION_BUS_ADDRESS: "+os.Getenv("DBUS_SESSION_BUS_ADDRESS"),
			"XDG_SESSION_TYPE: "+os.Getenv("XDG_SESSION_TYPE"),
		)
		if err == nil {
			// Best-effort cleanup of the sentinel entry. Delete errors are
			// ignored because Set already proved the keychain is functional.
			_ = keyring.Delete(keychain.ServiceName, "__obscuro_probe__")
			status.Available = true
			status.Reason = "ready"
			return status
		}
		status.Verbose = append(status.Verbose,
			"error: "+err.Error(),
			keychainRemediation().String(),
		)
		switch {
		case errors.Is(err, keyring.ErrUnsupportedPlatform):
			status.Reason = "unsupported platform"
		case strings.Contains(err.Error(), "The name org.freedesktop.secrets") ||
			strings.Contains(err.Error(), "dbus"):
			status.Reason = "dbus session bus unreachable — install gnome-keyring or use file backend"
		case strings.Contains(err.Error(), "unlock correct collection") ||
			strings.Contains(err.Error(), "locked"):
			status.Reason = "Secret Service reachable but login collection is locked/missing — try 'secret-tool store' first, or use file backend"
		default:
			status.Reason = fmt.Sprintf("keychain probe failed: %v", err)
		}
		return status
	}
}

// defaultFileProbe returns a closure that verifies the managed-file backend
// directory is writable. The salt is captured so the probe exercises the
// exact XDG path that Write/Read would use for this vault.
func defaultFileProbe(salt string) backendProbe {
	return func() BackendStatus {
		status := BackendStatus{
			Kind: BackendFile,
			Name: "managed file",
		}
		path, err := pwfile.Path(salt)
		status.Verbose = append(status.Verbose,
			"resolved path: "+path,
			"XDG_CONFIG_HOME: "+os.Getenv("XDG_CONFIG_HOME"),
			fmt.Sprintf("pwfile present: %t", pwfile.Exists(salt)),
		)
		if err != nil {
			status.Available = false
			status.Reason = fmt.Sprintf("XDG path unwritable: %v", err)
			return status
		}
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			status.Available = false
			status.Reason = fmt.Sprintf("XDG path unwritable: %v", err)
			return status
		}
		tmp := filepath.Join(dir, ".probe.tmp")
		if err := os.WriteFile(tmp, []byte(""), 0o600); err != nil {
			status.Available = false
			status.Reason = fmt.Sprintf("XDG path unwritable: %v", err)
			return status
		}
		_ = os.Remove(tmp)
		status.Available = true
		status.Reason = fmt.Sprintf("ready — writes to %s", path)
		return status
	}
}
