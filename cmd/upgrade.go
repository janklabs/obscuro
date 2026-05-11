package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/janklabs/obscuro/internal/version"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

const (
	repoOwner = "janklabs"
	repoName  = "obscuro"
)

var (
	apiReleasesURL = "https://api.github.com/repos/" + repoOwner + "/" + repoName + "/releases"
	apiLatestURL   = "https://api.github.com/repos/" + repoOwner + "/" + repoName + "/releases/latest"
	downloadBase   = "https://github.com/" + repoOwner + "/" + repoName + "/releases/download"
)

var upgradeSkipChecksum bool

var upgradeCmd = &cobra.Command{
	Use:          "upgrade",
	Short:        "Upgrade obscuro to the latest version",
	Long:         `Downloads the latest prebuilt release binary from GitHub and replaces the current binary.`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := runUpgrade(); err != nil {
			fmt.Fprintf(os.Stderr, "\nUpgrade failed. You can reinstall manually:\n  curl -sSL https://raw.githubusercontent.com/janklabs/obscuro/main/install.sh | sh\n")
			return err
		}
		return nil
	},
}

func runUpgrade() error {
	return runUpgradeFromURLs(version.Version, apiLatestURL, downloadBase, apiReleasesURL)
}

func runUpgradeFromURLs(currentVersion, latestTagAPIURL, downloadBaseURL, releasesAPIURL string) error {
	// Temporarily redirect package-level URLs so fetchLatestTag/fetchChangelog
	// (whose signatures are intentionally preserved) use the injected endpoints.
	prevLatest, prevDownload, prevReleases := apiLatestURL, downloadBase, apiReleasesURL
	apiLatestURL, downloadBase, apiReleasesURL = latestTagAPIURL, downloadBaseURL, releasesAPIURL
	defer func() {
		apiLatestURL, downloadBase, apiReleasesURL = prevLatest, prevDownload, prevReleases
	}()

	current := currentVersion
	fmt.Fprintf(os.Stderr, "Current version: %s\n", current)

	fmt.Fprintln(os.Stderr, "Fetching latest version...")
	latest, err := fetchLatestTag()
	if err != nil {
		return fmt.Errorf("fetching latest version: %w", err)
	}
	if latest == "" {
		return fmt.Errorf("no release tags found")
	}

	fmt.Fprintf(os.Stderr, "Latest version: %s\n", latest)

	if current != "dev" && semver.Compare(current, latest) >= 0 {
		fmt.Fprintf(os.Stderr, "Already up to date (%s)\n", current)
		return nil
	}

	if current != "dev" {
		if changelog := fetchChangelog(current, latest); changelog != "" {
			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(os.Stderr, changelog)
		}
	}

	assetName, err := assetNameFor(latest, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", "obscuro-upgrade-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	assetPath := filepath.Join(tmpDir, assetName)
	assetURL := fmt.Sprintf("%s/%s/%s", downloadBaseURL, latest, assetName)

	fmt.Fprintf(os.Stderr, "Downloading %s...\n", assetName)
	if err := downloadFile(assetURL, assetPath); err != nil {
		return fmt.Errorf("downloading binary: %w", err)
	}

	sumsURL := fmt.Sprintf("%s/%s/checksums.txt", downloadBaseURL, latest)
	sumsPath := filepath.Join(tmpDir, "checksums.txt")
	if err := downloadFile(sumsURL, sumsPath); err != nil {
		if upgradeSkipChecksum || os.Getenv("OBSCURO_INSECURE_SKIP_CHECKSUM") == "1" {
			fmt.Fprintln(os.Stderr, "warning: skipping checksum verification (OBSCURO_INSECURE_SKIP_CHECKSUM=1 / --insecure-skip-checksum)")
		} else {
			return fmt.Errorf("downloading checksums: %w (set --insecure-skip-checksum or OBSCURO_INSECURE_SKIP_CHECKSUM=1 to bypass; this is unsafe)", err)
		}
	} else {
		if err := verifyChecksum(assetPath, sumsPath, assetName); err != nil {
			return fmt.Errorf("verifying checksum: %w", err)
		}
		fmt.Fprintln(os.Stderr, "Checksum OK")
	}

	if err := os.Chmod(assetPath, 0o755); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding current binary: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("resolving symlinks: %w", err)
	}

	if err := atomicReplace(assetPath, execPath); err != nil {
		return fmt.Errorf("replacing binary: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Upgraded obscuro from %s to %s\n", current, latest)
	return nil
}

func init() {
	upgradeCmd.Flags().BoolVar(&upgradeSkipChecksum, "insecure-skip-checksum", false, "skip SHA-256 verification of the downloaded binary (UNSAFE)")
	rootCmd.AddCommand(upgradeCmd)
}

func assetNameFor(tag, goos, goarch string) (string, error) {
	switch goos {
	case "linux", "darwin", "windows":
	default:
		return "", fmt.Errorf("unsupported OS: %s", goos)
	}
	switch goarch {
	case "amd64", "arm64":
	default:
		return "", fmt.Errorf("unsupported architecture: %s", goarch)
	}
	ext := ""
	if goos == "windows" {
		ext = ".exe"
	}
	return fmt.Sprintf("obscuro-%s-%s-%s%s", tag, goos, goarch, ext), nil
}

func downloadFile(url, dst string) error {
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: %s", url, resp.Status)
	}

	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return err
	}
	return nil
}

func verifyChecksum(filePath, sumsPath, name string) error {
	data, err := os.ReadFile(sumsPath)
	if err != nil {
		return err
	}
	var expected string
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == name {
			expected = fields[0]
			break
		}
	}
	if expected == "" {
		return fmt.Errorf("no checksum entry for %s", name)
	}

	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	actual := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("checksum mismatch: expected=%s actual=%s", expected, actual)
	}
	return nil
}

// fetchChangelog returns a formatted changelog for releases between current
// (exclusive) and latest (inclusive). Returns an empty string on any error so
// upgrades are never blocked by changelog issues.
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

func fetchLatestTag() (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(apiLatestURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: %s", apiLatestURL, resp.Status)
	}

	var rel struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", err
	}
	if !semver.IsValid(rel.TagName) {
		return "", fmt.Errorf("invalid tag from GitHub: %q", rel.TagName)
	}
	return rel.TagName, nil
}

// atomicReplace replaces dst with src by writing a temp file beside dst and
// renaming. Same-filesystem rename is required for atomicity.
func atomicReplace(src, dst string) error {
	srcData, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	tmpFile := dst + ".tmp"
	if err := os.WriteFile(tmpFile, srcData, 0o755); err != nil {
		return err
	}

	if err := os.Rename(tmpFile, dst); err != nil {
		os.Remove(tmpFile)
		return err
	}
	return nil
}
