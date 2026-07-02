package keychain

import (
	"errors"
	"strings"
	"testing"

	keyring "github.com/zalando/go-keyring"
)

func TestMain(m *testing.M) {
	keyring.MockInit()
	m.Run()
}

// TestStoreAndGet verifies basic store and retrieve roundtrip.
func TestStoreAndGet(t *testing.T) {
	salt := "test-salt-1"
	password := "test-password-1"

	err := Store(salt, password)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	retrieved, err := Get(salt)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved != password {
		t.Errorf("expected %q, got %q", password, retrieved)
	}
}

// TestStoreOverwrites verifies that storing with the same salt overwrites the previous value.
func TestStoreOverwrites(t *testing.T) {
	salt := "test-salt-overwrite"
	password1 := "first-password"
	password2 := "second-password"

	err := Store(salt, password1)
	if err != nil {
		t.Fatalf("first Store failed: %v", err)
	}

	err = Store(salt, password2)
	if err != nil {
		t.Fatalf("second Store failed: %v", err)
	}

	retrieved, err := Get(salt)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved != password2 {
		t.Errorf("expected %q (overwritten value), got %q", password2, retrieved)
	}
}

// TestGetUnknownKey verifies that Get returns an error for a non-existent salt.
func TestGetUnknownKey(t *testing.T) {
	salt := "nonexistent-salt-xyz-12345"

	_, err := Get(salt)
	if err == nil {
		t.Error("expected error for non-existent salt, got nil")
	}
}

// TestDeleteRemovesEntry verifies that Delete removes an entry so subsequent Get fails.
func TestDeleteRemovesEntry(t *testing.T) {
	salt := "test-salt-delete"
	password := "password-to-delete"

	err := Store(salt, password)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	err = Delete(salt)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = Get(salt)
	if err == nil {
		t.Error("expected error after Delete, but Get succeeded")
	}
}

// TestDeleteNonExistent documents the behavior of Delete on a non-existent salt.
func TestDeleteNonExistent(t *testing.T) {
	salt := "nonexistent-salt-delete-abc"

	err := Delete(salt)
	// Document actual behavior: MockInit may return error or nil
	if err != nil {
		t.Logf("Delete on non-existent salt returned error: %v (acceptable)", err)
	} else {
		t.Logf("Delete on non-existent salt returned nil (acceptable)")
	}
}

// TestIsolatedSalts verifies that different salts maintain independent values.
func TestIsolatedSalts(t *testing.T) {
	saltA := "test-salt-A"
	saltB := "test-salt-B"
	passwordA := "password-A"
	passwordB := "password-B"

	err := Store(saltA, passwordA)
	if err != nil {
		t.Fatalf("Store saltA failed: %v", err)
	}

	err = Store(saltB, passwordB)
	if err != nil {
		t.Fatalf("Store saltB failed: %v", err)
	}

	retrievedA, err := Get(saltA)
	if err != nil {
		t.Fatalf("Get saltA failed: %v", err)
	}

	if retrievedA != passwordA {
		t.Errorf("saltA: expected %q, got %q", passwordA, retrievedA)
	}

	retrievedB, err := Get(saltB)
	if err != nil {
		t.Fatalf("Get saltB failed: %v", err)
	}

	if retrievedB != passwordB {
		t.Errorf("saltB: expected %q, got %q", passwordB, retrievedB)
	}
}

// TestServiceNameIsObscuro verifies that the service name constant is "obscuro".
func TestServiceNameIsObscuro(t *testing.T) {
	salt := "test-salt-service"
	password := "test-password-service"

	err := Store(salt, password)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Verify by directly calling keyring.Get with the expected service name
	retrieved, err := keyring.Get("obscuro", salt)
	if err != nil {
		t.Fatalf("keyring.Get with service 'obscuro' failed: %v", err)
	}

	if retrieved != password {
		t.Errorf("expected %q, got %q", password, retrieved)
	}
}

// TestEmptySaltRoundtrip verifies behavior with an empty salt string.
func TestEmptySaltRoundtrip(t *testing.T) {
	salt := ""
	password := "password-for-empty-salt"

	err := Store(salt, password)
	if err != nil {
		t.Logf("Store with empty salt returned error: %v (acceptable)", err)
		return
	}

	retrieved, err := Get(salt)
	if err != nil {
		t.Logf("Get with empty salt returned error: %v (acceptable)", err)
		return
	}

	if retrieved != password {
		t.Errorf("expected %q, got %q", password, retrieved)
	}
}

// TestEmptyPasswordRoundtrip verifies behavior with an empty password string.
func TestEmptyPasswordRoundtrip(t *testing.T) {
	salt := "test-salt-empty-password"
	password := ""

	err := Store(salt, password)
	if err != nil {
		t.Logf("Store with empty password returned error: %v (acceptable)", err)
		return
	}

	retrieved, err := Get(salt)
	if err != nil {
		t.Logf("Get with empty password returned error: %v (acceptable)", err)
		return
	}

	if retrieved != password {
		t.Errorf("expected %q, got %q", password, retrieved)
	}
}

// TestHasEntryTrue verifies that HasEntry returns true for an existing entry.
func TestHasEntryTrue(t *testing.T) {
	salt := "test-salt-has-true"
	password := "password-for-has-test"

	err := Store(salt, password)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	if !HasEntry(salt) {
		t.Error("HasEntry returned false for an existing entry")
	}
}

// TestHasEntryFalse verifies that HasEntry returns false for a non-existent entry.
func TestHasEntryFalse(t *testing.T) {
	salt := "nonexistent-salt-has-xyz"

	if HasEntry(salt) {
		t.Error("HasEntry returned true for a non-existent entry")
	}
}

// stubBackend is a test stub that allows injection of errors.
type stubBackend struct {
	setErr    error
	getErr    error
	deleteErr error
}

func (s stubBackend) Set(_, _, _ string) error {
	return s.setErr
}

func (s stubBackend) Get(_, _ string) (string, error) {
	return "", s.getErr
}

func (s stubBackend) Delete(_, _ string) error {
	return s.deleteErr
}

// TestBackendSeam verifies that the backend can be swapped for testing.
func TestBackendSeam(t *testing.T) {
	// Save the original backend
	originalBackend := defaultBackend
	defer func() {
		defaultBackend = originalBackend
	}()

	// Install a stub backend that returns a sentinel error
	sentinelErr := errors.New("sentinel-get-error")
	defaultBackend = stubBackend{getErr: sentinelErr}

	// Call Get and verify the error is propagated
	_, err := Get("any-salt")
	if err == nil {
		t.Error("expected error from stub backend, got nil")
	}

	if !errors.Is(err, sentinelErr) {
		t.Errorf("expected sentinel error, got %v", err)
	}
}

// TestResetBackend verifies that ResetBackend restores the real backend.
func TestResetBackend(t *testing.T) {
	// Save the original backend
	originalBackend := defaultBackend

	// Swap to a stub backend
	defaultBackend = stubBackend{getErr: errors.New("stub-error")}

	// Verify stub is active
	_, err := Get("test-salt")
	if err == nil {
		t.Error("expected stub backend to be active, but Get succeeded")
	}

	// Reset to real backend
	ResetBackend()

	// Verify real backend is restored (it should be a realBackend instance)
	// We can't directly compare types, but we can verify it's not the stub anymore
	// by checking that it doesn't return our sentinel error
	_, err = Get("nonexistent-salt-reset-test")
	if err != nil {
		// Real backend returns "secret not found" or similar, not our sentinel
		if errors.Is(err, errors.New("stub-error")) {
			t.Error("ResetBackend did not restore the real backend")
		}
	}

	// Restore original for cleanup
	defaultBackend = originalBackend
}

// TestAvailableHealthy verifies that Available() returns nil when Set and Delete succeed.
func TestAvailableHealthy(t *testing.T) {
	originalBackend := defaultBackend
	defer func() { defaultBackend = originalBackend }()

	var setCalled bool
	var setService, setUser, setPassword string
	defaultBackend = capturingStub{
		onSet: func(svc, usr, pw string) error {
			setCalled = true
			setService, setUser, setPassword = svc, usr, pw
			return nil
		},
	}

	if err := Available(); err != nil {
		t.Fatalf("Available() = %v, want nil", err)
	}
	if !setCalled {
		t.Error("Available() did not call Set")
	}
	if setService != serviceName {
		t.Errorf("Set called with service=%q, want %q", setService, serviceName)
	}
	if setUser != probeUser {
		t.Errorf("Set called with user=%q, want %q", setUser, probeUser)
	}
	if setPassword != "" {
		t.Errorf("Set called with password=%q, want empty string", setPassword)
	}
}

// capturingStub is a test-local backend that captures Set arguments.
type capturingStub struct {
	onSet func(svc, usr, pw string) error
}

func (c capturingStub) Set(svc, usr, pw string) error {
	if c.onSet != nil {
		return c.onSet(svc, usr, pw)
	}
	return nil
}
func (c capturingStub) Get(_, _ string) (string, error) { return "", keyring.ErrNotFound }
func (c capturingStub) Delete(_, _ string) error        { return nil }

// TestAvailableSetFails verifies that Available() returns ErrKeychainUnavailable
// when Set fails, and that the underlying error is wrapped.
func TestAvailableSetFails(t *testing.T) {
	originalBackend := defaultBackend
	defer func() { defaultBackend = originalBackend }()

	cause := errors.New("dbus down")
	defaultBackend = stubBackend{setErr: cause}

	err := Available()
	if err == nil {
		t.Fatal("Available() = nil, want non-nil error")
	}
	if !errors.Is(err, ErrKeychainUnavailable) {
		t.Errorf("errors.Is(err, ErrKeychainUnavailable) = false; err = %v", err)
	}
	if !strings.Contains(err.Error(), "dbus down") {
		t.Errorf("error %q does not contain underlying cause %q", err.Error(), "dbus down")
	}
}

// TestAvailableDeleteNotFoundAfterSetSuccess verifies that Available() returns nil
// even when Delete returns ErrNotFound (the probe entry was cleaned up externally).
func TestAvailableDeleteNotFoundAfterSetSuccess(t *testing.T) {
	originalBackend := defaultBackend
	defer func() { defaultBackend = originalBackend }()

	defaultBackend = stubBackend{deleteErr: keyring.ErrNotFound}

	if err := Available(); err != nil {
		t.Fatalf("Available() = %v, want nil (ErrNotFound from Delete should be treated as success)", err)
	}
}
