package cmd

import (
	"bufio"
	"bytes"
	"os"
	"runtime"
	"strings"
)

// OSInfo holds the detected OS and distro information for the current host.
type OSInfo struct {
	Platform   string            // "linux", "darwin", "windows", "other"
	Distro     string            // e.g. "ubuntu", "debian", "fedora", "arch", "alpine", "nixos", "macos", "windows", "other"
	DistroLike string            // from ID_LIKE= in /etc/os-release (Linux only), e.g. "debian" for Ubuntu
	VersionID  string            // e.g. "24.04", "40", "11"
	PrettyName string            // e.g. "Ubuntu 24.04.4 LTS"
	Arch       string            // runtime.GOARCH value
	Extra      map[string]string // per-platform extras: macOS "homebrew"="apple-silicon"|"intel"|"macports"|"none"; Windows "shell"="pwsh"|"powershell"|"none"
}

// osDetectFn is the OS detection function; swapped in tests.
// Mirrors the promptPasswordFn seam at cmd/root.go.
var osDetectFn func() OSInfo = detectOS

// detectOS returns OS/distro information for the current host.
// It reads /etc/os-release on Linux and probes filesystem paths on macOS/Windows.
// No external process is spawned; only stdlib is used.
func detectOS() OSInfo {
	info := OSInfo{
		Arch:  runtime.GOARCH,
		Extra: map[string]string{},
	}
	switch runtime.GOOS {
	case "linux":
		info.Platform = "linux"
		data, err := os.ReadFile("/etc/os-release")
		if err == nil {
			info.Distro, info.DistroLike, info.VersionID, info.PrettyName = parseOSRelease(data)
		}
		if info.Distro == "" {
			info.Distro = "other"
		}
	case "darwin":
		info.Platform = "darwin"
		info.Distro = "macos"
		info.Extra = detectMacInfo()
	case "windows":
		info.Platform = "windows"
		info.Distro = "windows"
		info.Extra = detectWindowsInfo()
	default:
		info.Platform = "other"
		info.Distro = "other"
	}
	return info
}

// parseOSRelease parses the contents of /etc/os-release and extracts
// the distro ID, ID_LIKE, VERSION_ID, and PRETTY_NAME fields.
// Values are unquoted (double or single quotes stripped).
func parseOSRelease(data []byte) (distro, distroLike, versionID, prettyName string) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue
		}
		key := line[:idx]
		val := unquoteOSReleaseValue(line[idx+1:])
		switch key {
		case "ID":
			distro = strings.ToLower(val)
		case "ID_LIKE":
			distroLike = val
		case "VERSION_ID":
			versionID = val
		case "PRETTY_NAME":
			prettyName = val
		}
	}
	return
}

// unquoteOSReleaseValue strips surrounding double or single quotes from a value.
func unquoteOSReleaseValue(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// detectMacInfo probes the filesystem for Homebrew and MacPorts installations.
func detectMacInfo() map[string]string {
	return detectMacInfoFrom(func(p string) bool {
		_, err := os.Stat(p)
		return err == nil
	})
}

// detectMacInfoFrom is the pure helper for detectMacInfo; injected for tests.
func detectMacInfoFrom(exists func(string) bool) map[string]string {
	extra := map[string]string{}
	switch {
	case exists("/opt/homebrew/bin/brew"):
		extra["homebrew"] = "apple-silicon"
	case exists("/usr/local/bin/brew"):
		extra["homebrew"] = "intel"
	case exists("/opt/local/bin/port"):
		extra["homebrew"] = "macports"
	default:
		extra["homebrew"] = "none"
	}
	return extra
}

// detectWindowsInfo probes for PowerShell versions using the package-level lookPath seam.
func detectWindowsInfo() map[string]string {
	return detectWindowsInfoFrom(lookPath)
}

// detectWindowsInfoFrom is the pure helper for detectWindowsInfo; injected for tests.
func detectWindowsInfoFrom(lookPath func(string) (string, error)) map[string]string {
	extra := map[string]string{}
	if _, err := lookPath("pwsh"); err == nil {
		extra["shell"] = "pwsh"
	} else if _, err := lookPath("powershell"); err == nil {
		extra["shell"] = "powershell"
	} else {
		extra["shell"] = "none"
	}
	return extra
}
