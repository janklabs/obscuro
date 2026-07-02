package cmd

import "runtime"

// keychainRemediation returns a platform-specific hint pointing to
// alternatives when the OS keychain is unavailable. Format follows the
// repo error-voice convention: lowercase first word, no trailing period,
// em-dash separator, single-quoted command names.
//
// FreeBSD/NetBSD/OpenBSD are not release targets (matrix is
// linux/darwin/windows); they fall to the generic message.
func keychainRemediation() string {
	switch runtime.GOOS {
	case "linux":
		return "keychain unavailable — install gnome-keyring or KeePassXC with secret-service, or use '--password-file' / OBSCURO_PASSWORD env"
	case "darwin":
		return "keychain unavailable — open Keychain Access and unlock the login keychain, or use '--password-file' / OBSCURO_PASSWORD env"
	case "windows":
		return "keychain unavailable — check Credential Manager is accessible, or use '--password-file' / OBSCURO_PASSWORD env"
	default:
		return "keychain unavailable on this platform — use '--password-file' / OBSCURO_PASSWORD env"
	}
}
