package cmd

import (
	"strings"
	"testing"
)

func TestSet_printsConfirmation_whenValueFlagSuppliesSecret(t *testing.T) {
	setup(t)
	initVault(t)
	resetSetFlags()

	_, stderr, err := execCmd(t, "set", "API_KEY", "--password", testPassword, "--value", "my-secret-123")
	if err != nil {
		t.Fatalf("set failed: %v", err)
	}
	if !strings.Contains(stderr, "Secret 'API_KEY' was set.") {
		t.Fatalf("expected set confirmation in stderr, got %q", stderr)
	}
}

func TestSet_printsEntryNotice_whenPromptingForSecret(t *testing.T) {
	setup(t)
	initVault(t)
	resetSetFlags()
	withFakePassword(t, "interactive-value")

	_, stderr, err := execCmd(t, "set", "IKEY", "--password", testPassword)
	if err != nil {
		t.Fatalf("set failed: %v", err)
	}
	if !strings.Contains(stderr, "Enter the secret value for 'IKEY'.") {
		t.Fatalf("expected interactive entry notice in stderr, got %q", stderr)
	}
	if !strings.Contains(stderr, "Secret 'IKEY' was set.") {
		t.Fatalf("expected set confirmation in stderr, got %q", stderr)
	}
}
