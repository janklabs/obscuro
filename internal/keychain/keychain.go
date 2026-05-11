package keychain

import (
	"github.com/zalando/go-keyring"
)

const serviceName = "obscuro"

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

// Store saves the password in the OS keychain, keyed by the vault's salt.
func Store(salt, password string) error {
	return defaultBackend.Set(serviceName, salt, password)
}

// Get retrieves the password from the OS keychain for the given salt.
func Get(salt string) (string, error) {
	return defaultBackend.Get(serviceName, salt)
}

// Delete removes the password from the OS keychain for the given salt.
func Delete(salt string) error {
	return defaultBackend.Delete(serviceName, salt)
}

// HasEntry returns true if a keychain entry exists for the given salt.
func HasEntry(salt string) bool {
	_, err := defaultBackend.Get(serviceName, salt)
	return err == nil
}
