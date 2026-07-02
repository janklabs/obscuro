package cmd

import (
	"errors"
	"testing"
)

func TestParseOSRelease(t *testing.T) {
	cases := []struct {
		name       string
		input      string
		wantDistro string
		wantLike   string
		wantVer    string
	}{
		{"ubuntu-24.04", "PRETTY_NAME=\"Ubuntu 24.04.4 LTS\"\nID=ubuntu\nID_LIKE=debian\nVERSION_ID=\"24.04\"\n", "ubuntu", "debian", "24.04"},
		{"debian-12", "ID=debian\nVERSION_ID=\"12\"\nPRETTY_NAME=\"Debian GNU/Linux 12 (bookworm)\"\n", "debian", "", "12"},
		{"fedora-40", "ID=fedora\nVERSION_ID=40\nPRETTY_NAME=\"Fedora Linux 40 (Workstation Edition)\"\n", "fedora", "", "40"},
		{"rhel-9", "ID=\"rhel\"\nID_LIKE=\"fedora\"\nVERSION_ID=\"9.4\"\n", "rhel", "fedora", "9.4"},
		{"rocky-9", "ID=\"rocky\"\nID_LIKE=\"rhel centos fedora\"\nVERSION_ID=\"9.4\"\n", "rocky", "rhel centos fedora", "9.4"},
		{"opensuse", "ID=\"opensuse-leap\"\nID_LIKE=\"suse\"\nVERSION_ID=\"15.6\"\n", "opensuse-leap", "suse", "15.6"},
		{"arch", "ID=arch\nPRETTY_NAME=\"Arch Linux\"\n", "arch", "", ""},
		{"alpine", "ID=alpine\nVERSION_ID=3.20.0\n", "alpine", "", "3.20.0"},
		{"empty", "", "", "", ""},
		{"comments-only", "# comment\n# another\n", "", "", ""},
		{"malformed-noquote", "ID=ubuntu\nVERSION_ID=24.04\n", "ubuntu", "", "24.04"},
		{"quoted-values", "ID=\"fedora\"\nVERSION_ID=\"40\"\n", "fedora", "", "40"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			distro, distroLike, versionID, _ := parseOSRelease([]byte(tc.input))
			if distro != tc.wantDistro {
				t.Errorf("distro: got %q, want %q", distro, tc.wantDistro)
			}
			if distroLike != tc.wantLike {
				t.Errorf("distroLike: got %q, want %q", distroLike, tc.wantLike)
			}
			if versionID != tc.wantVer {
				t.Errorf("versionID: got %q, want %q", versionID, tc.wantVer)
			}
		})
	}
}

func TestDetectMacInfoFrom(t *testing.T) {
	cases := []struct {
		name         string
		paths        map[string]bool
		wantHomebrew string
	}{
		{"apple-silicon", map[string]bool{"/opt/homebrew/bin/brew": true}, "apple-silicon"},
		{"intel", map[string]bool{"/usr/local/bin/brew": true}, "intel"},
		{"macports", map[string]bool{"/opt/local/bin/port": true}, "macports"},
		{"none", map[string]bool{}, "none"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			existsFn := func(p string) bool { return tc.paths[p] }
			extra := detectMacInfoFrom(existsFn)
			if extra["homebrew"] != tc.wantHomebrew {
				t.Errorf("homebrew: got %q, want %q", extra["homebrew"], tc.wantHomebrew)
			}
		})
	}
}

func TestDetectWindowsInfoFrom(t *testing.T) {
	cases := []struct {
		name       string
		pwsh       bool
		powershell bool
		wantShell  string
	}{
		{"pwsh-only", true, false, "pwsh"},
		{"pwsh-and-ps5", true, true, "pwsh"},
		{"ps5-only", false, true, "powershell"},
		{"none", false, false, "none"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lookPathFn := func(name string) (string, error) {
				if name == "pwsh" && tc.pwsh {
					return "/usr/bin/pwsh", nil
				}
				if name == "powershell" && tc.powershell {
					return "/usr/bin/powershell", nil
				}
				return "", errors.New("not found")
			}
			extra := detectWindowsInfoFrom(lookPathFn)
			if extra["shell"] != tc.wantShell {
				t.Errorf("shell: got %q, want %q", extra["shell"], tc.wantShell)
			}
		})
	}
}
