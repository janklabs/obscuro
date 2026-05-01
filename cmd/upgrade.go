package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/janklabs/obscuro/internal/version"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

const repoURL = "https://github.com/janklabs/obscuro.git"

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade obscuro to the latest version",
	Long: `Fetches the latest release tag from GitHub, builds from source,
and replaces the current binary. Requires Go to be installed.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := runUpgrade(); err != nil {
			fmt.Fprintf(os.Stderr, "\nUpgrade failed. You can reinstall manually:\n  curl -sSL https://raw.githubusercontent.com/janklabs/obscuro/main/install.sh | bash\n")
			return err
		}
		return nil
	},
}

func runUpgrade() error {
	current := version.Version
	fmt.Fprintf(os.Stderr, "Current version: %s\n", current)

	// Fetch latest tag
	fmt.Fprintln(os.Stderr, "Fetching latest version...")
	latest, err := fetchLatestTag()
	if err != nil {
		return fmt.Errorf("fetching latest version: %w", err)
	}
	if latest == "" {
		return fmt.Errorf("no release tags found")
	}

	fmt.Fprintf(os.Stderr, "Latest version: %s\n", latest)

	// Compare versions
	if current != "dev" && semver.Compare(current, latest) >= 0 {
		fmt.Fprintf(os.Stderr, "Already up to date (%s)\n", current)
		return nil
	}

	// Print changelog between versions.
	if current != "dev" {
		if changelog := fetchChangelog(current, latest); changelog != "" {
			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(os.Stderr, changelog)
		}
	}

	// Clone and build
	fmt.Fprintf(os.Stderr, "Downloading and building %s...\n", latest)
	tmpDir, err := os.MkdirTemp("", "obscuro-upgrade-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	cloneCmd := exec.Command("git", "-c", "advice.detachedHead=false", "clone", "--quiet", "--depth", "1", "--branch", latest, repoURL, filepath.Join(tmpDir, "obscuro"))
	cloneCmd.Stderr = os.Stderr
	if err := cloneCmd.Run(); err != nil {
		return fmt.Errorf("cloning repo: %w", err)
	}

	ldflags := fmt.Sprintf("-X github.com/janklabs/obscuro/internal/version.Version=%s", latest)
	binaryName := "obscuro"
	if runtime.GOOS == "windows" {
		binaryName = "obscuro.exe"
	}
	newBinary := filepath.Join(tmpDir, "bin-"+binaryName)

	buildCmd := exec.Command("go", "build", "-ldflags", ldflags, "-o", newBinary, ".")
	buildCmd.Dir = filepath.Join(tmpDir, "obscuro")
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("building: %w", err)
	}

	// Replace current binary
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding current binary: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("resolving symlinks: %w", err)
	}

	if err := atomicReplace(newBinary, execPath); err != nil {
		return fmt.Errorf("replacing binary: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Upgraded obscuro from %s to %s\n", current, latest)
	return nil
}

func init() {
	rootCmd.AddCommand(upgradeCmd)
}

const apiReleasesURL = "https://api.github.com/repos/janklabs/obscuro/releases"

// fetchChangelog returns a formatted changelog string for all releases
// between current (exclusive) and latest (inclusive). Returns an empty
// string on any error so that upgrades are never blocked.
func fetchChangelog(current, latest string) string {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(apiReleasesURL)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var releases []struct {
		TagName string `json:"tag_name"`
		Name    string `json:"name"`
		Body    string `json:"body"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return ""
	}

	// Filter to releases between current and latest.
	var relevant []struct {
		tag  string
		body string
	}
	for _, r := range releases {
		if !semver.IsValid(r.TagName) {
			continue
		}
		if semver.Compare(r.TagName, current) > 0 && semver.Compare(r.TagName, latest) <= 0 {
			body := strings.TrimSpace(r.Body)
			if body == "" {
				body = "(no release notes)"
			}
			relevant = append(relevant, struct {
				tag  string
				body string
			}{tag: r.TagName, body: body})
		}
	}

	if len(relevant) == 0 {
		return ""
	}

	// Sort oldest first.
	sort.Slice(relevant, func(i, j int) bool {
		return semver.Compare(relevant[i].tag, relevant[j].tag) < 0
	})

	var b strings.Builder
	b.WriteString("Changelog:\n")
	for _, r := range relevant {
		b.WriteString(fmt.Sprintf("\n  %s\n", r.tag))
		for _, line := range strings.Split(r.body, "\n") {
			b.WriteString(fmt.Sprintf("    %s\n", line))
		}
	}
	return b.String()
}

// fetchLatestTag returns the highest semver tag from the remote repo.
func fetchLatestTag() (string, error) {
	out, err := exec.Command("git", "ls-remote", "--tags", "--refs", repoURL).Output()
	if err != nil {
		return "", err
	}

	var tags []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "/")
		tag := parts[len(parts)-1]
		if semver.IsValid(tag) {
			tags = append(tags, tag)
		}
	}

	if len(tags) == 0 {
		return "", nil
	}

	sort.Slice(tags, func(i, j int) bool {
		return semver.Compare(tags[i], tags[j]) < 0
	})

	return tags[len(tags)-1], nil
}

// atomicReplace replaces dst with src by writing to a temp file and renaming.
func atomicReplace(src, dst string) error {
	srcFile, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	// Write next to the destination to ensure same filesystem for rename
	tmpFile := dst + ".tmp"
	if err := os.WriteFile(tmpFile, srcFile, 0o755); err != nil {
		return err
	}

	if err := os.Rename(tmpFile, dst); err != nil {
		os.Remove(tmpFile)
		return err
	}

	return nil
}
