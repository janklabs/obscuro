package keychain

import (
	"errors"
	"fmt"

	keyring "github.com/zalando/go-keyring"
)

// ServiceName is the OS keychain "service" identifier used by all
// obscuro entries. Exported so cmd/backend_detect.go can probe with
// the exact string without duplication.
const ServiceName = "obscuro"

// ErrKeychainUnavailable is returned by Available when the OS keychain cannot
// be reached on this host (dbus missing, Secret Service not running, etc.).
var ErrKeychainUnavailable = errors.New("keychain unavailable")

const probeUser = "__obscuro_probe__"

// backend is the interface for keychain operations, allowing test injection.
type backend interface {
	Set(service, user, password string) error
	Get(service, user string) (string, error)
	Delete(service, user string) error
}

// realBackend implements backend using the OS keychain.
type realBackend struct{}

func (realBackend) Set(service, user, password string) error {
	return keyring.Set(service, user, password)
}

func (realBackend) Get(service, user string) (string, error) {
	return keyring.Get(service, user)
}

func (realBackend) Delete(service, user string) error {
	return keyring.Delete(service, user)
}

// defaultBackend is the active backend; can be swapped for testing.
var defaultBackend backend = realBackend{}

// ResetBackend resets the backend to the real implementation (for testing).
func ResetBackend() {
	defaultBackend = realBackend{}
}

// ExportSetBackend installs a stub backend that returns the given errors.
// Intended ONLY for tests in other packages (cmd/); production callers must not use this.
func ExportSetBackend(setErr, deleteErr error) {
	defaultBackend = stubExport{setErr: setErr, deleteErr: deleteErr}
}

type stubExport struct {
	setErr    error
	deleteErr error
}

func (s stubExport) Set(_, _, _ string) error { return s.setErr }

// Get returns ErrNotFound so HasEntry correctly reports "no entry"
// during failure injection; tests that need a stored value should use
// useMockKeyring(t) (real MockInit) instead of this stub.
func (s stubExport) Get(_, _ string) (string, error) { return "", keyring.ErrNotFound }
func (s stubExport) Delete(_, _ string) error        { return s.deleteErr }

// Store saves the password in the OS keychain, keyed by the vault's salt.
func Store(salt, password string) error {
	return defaultBackend.Set(ServiceName, salt, password)
}

// Get retrieves the password from the OS keychain for the given salt.
func Get(salt string) (string, error) {
	return defaultBackend.Get(ServiceName, salt)
}

// Delete removes the password from the OS keychain for the given salt.
func Delete(salt string) error {
	return defaultBackend.Delete(ServiceName, salt)
}

// HasEntry returns true if a keychain entry exists for the given salt.
func HasEntry(salt string) bool {
	_, err := defaultBackend.Get(ServiceName, salt)
	return err == nil
}

// Available reports whether the OS keychain is usable on this host.
// It performs a trial Set + best-effort Delete of a well-known sentinel entry.
// Returns nil on success. Returns ErrKeychainUnavailable (wrapping the underlying
// error) if Set fails. Delete failures with keyring.ErrNotFound are treated
// as success (the probe was cleaned up or never persisted).
func Available() error {
	if err := defaultBackend.Set(ServiceName, probeUser, ""); err != nil {
		return fmt.Errorf("%w: %v", ErrKeychainUnavailable, err)
	}
	if err := defaultBackend.Delete(ServiceName, probeUser); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		// Best-effort cleanup. A non-ErrNotFound error here is logged only
		// in tests via the stub; keychain is provably functional (Set succeeded).
		_ = err
	}
	return nil
}
