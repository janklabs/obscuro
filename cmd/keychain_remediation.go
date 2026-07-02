package cmd

import (
	"fmt"
	"strings"
)

const docsURL = "https://obscuro.dev/docs/troubleshooting"

// Remediation holds a structured keychain-unavailable message.
type Remediation struct {
	Summary string   // greppable single-line summary, e.g. "keychain unavailable"
	Steps   []string // ordered action steps; step 1 = install, 2-3 = alternatives
	DocsURL string   // canonical docs URL
}

// String renders the full multi-line message for CLI display.
// Format: summary, blank line, numbered steps, blank line, docs link.
func (r Remediation) String() string {
	var b strings.Builder
	b.WriteString(r.Summary)
	b.WriteString("\n")
	for i, step := range r.Steps {
		fmt.Fprintf(&b, "\n%d. %s", i+1, step)
	}
	if r.DocsURL != "" {
		fmt.Fprintf(&b, "\n\nsee %s", r.DocsURL)
	}
	return b.String()
}

// Error returns the greppable single-line summary. Suitable for
// fmt.Errorf("%s: %w", r.Error(), err) where a compact log line is preferred.
func (r Remediation) Error() string {
	return r.Summary
}

// keychainRemediation detects the current OS/distro and returns a
// platform-specific Remediation with an exact install command.
func keychainRemediation() Remediation {
	info := osDetectFn()
	install := installStep(info)
	return Remediation{
		Summary: "keychain unavailable",
		Steps: []string{
			install,
			"use '--password-file /path/to/pw' (mode 600 recommended)",
			"use 'OBSCURO_PASSWORD=...' env var",
		},
		DocsURL: docsURL,
	}
}

// installStep returns the platform-specific install/fix action for the given OSInfo.
func installStep(info OSInfo) string {
	switch info.Platform {
	case "linux":
		return linuxInstallStep(info.Distro, info.DistroLike)
	case "darwin":
		return darwinInstallStep(info.Extra["homebrew"])
	case "windows":
		return windowsInstallStep(info.Extra["shell"])
	default:
		return "keychain not supported on this platform"
	}
}

func linuxInstallStep(distro, distroLike string) string {
	switch distro {
	case "ubuntu", "debian", "linuxmint", "pop":
		return "install gnome-keyring: sudo apt-get install gnome-keyring"
	case "fedora", "rhel", "centos", "rocky", "almalinux":
		return "install gnome-keyring: sudo dnf install gnome-keyring"
	case "opensuse", "opensuse-leap", "opensuse-tumbleweed", "sles":
		return "install gnome-keyring: sudo zypper install gnome-keyring"
	case "arch", "manjaro", "endeavouros", "garuda":
		return "install gnome-keyring: sudo pacman -S gnome-keyring"
	case "alpine":
		return "install gnome-keyring: sudo apk add gnome-keyring"
	case "nixos":
		return "install gnome-keyring: add 'services.gnome.gnome-keyring.enable = true;' to configuration.nix"
	default:
		// Try to use ID_LIKE for downstream distros
		return linuxFallbackStep(distroLike)
	}
}

func linuxFallbackStep(distroLike string) string {
	like := strings.ToLower(distroLike)
	switch {
	case strings.Contains(like, "ubuntu") || strings.Contains(like, "debian"):
		return "install gnome-keyring: sudo apt-get install gnome-keyring"
	case strings.Contains(like, "fedora") || strings.Contains(like, "rhel"):
		return "install gnome-keyring: sudo dnf install gnome-keyring"
	case strings.Contains(like, "suse"):
		return "install gnome-keyring: sudo zypper install gnome-keyring"
	case strings.Contains(like, "arch"):
		return "install gnome-keyring: sudo pacman -S gnome-keyring"
	default:
		return "install gnome-keyring or KeePassXC with secret-service (see docs)"
	}
}

func darwinInstallStep(homebrew string) string {
	switch homebrew {
	case "apple-silicon":
		return "unlock login keychain via Keychain Access, or reinstall: /opt/homebrew/bin/brew reinstall gnome-keyring"
	case "intel":
		return "unlock login keychain via Keychain Access, or reinstall: /usr/local/bin/brew reinstall gnome-keyring"
	case "macports":
		return "unlock login keychain via Keychain Access; MacPorts users: sudo port install gnome-keyring"
	default:
		return "unlock login keychain via Keychain Access"
	}
}

func windowsInstallStep(shell string) string {
	switch shell {
	case "pwsh":
		return "verify Credential Manager: pwsh -Command \"Get-Command cmdkey\""
	case "powershell":
		return "verify Credential Manager: powershell -Command \"Get-Command cmdkey\""
	default:
		return "verify Credential Manager is accessible via Control Panel > Credential Manager"
	}
}
