package keychain

import (
	"github.com/zalando/go-keyring"
)

const serviceName = "obscuro"

// Store saves the password in the OS keychain, keyed by the vault's salt.
func Store(salt, password string) error {
	return keyring.Set(serviceName, salt, password)
}

// Get retrieves the password from the OS keychain for the given salt.
func Get(salt string) (string, error) {
	return keyring.Get(serviceName, salt)
}

// Delete removes the password from the OS keychain for the given salt.
func Delete(salt string) error {
	return keyring.Delete(serviceName, salt)
}

// HasEntry returns true if a keychain entry exists for the given salt.
func HasEntry(salt string) bool {
	_, err := keyring.Get(serviceName, salt)
	return err == nil
}
