package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

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

		// Clone and build
		fmt.Fprintf(os.Stderr, "Downloading and building %s...\n", latest)
		tmpDir, err := os.MkdirTemp("", "obscuro-upgrade-*")
		if err != nil {
			return fmt.Errorf("creating temp dir: %w", err)
		}
		defer os.RemoveAll(tmpDir)

		cloneCmd := exec.Command("git", "clone", "--depth", "1", "--branch", latest, repoURL, filepath.Join(tmpDir, "obscuro"))
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
	},
}

func init() {
	rootCmd.AddCommand(upgradeCmd)
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
