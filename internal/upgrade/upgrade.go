// Package upgrade replaces the running voban executable with the latest
// release published on GitHub, after verifying its SHA-256 checksum.
package upgrade

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/vobanai/voban-cli/internal/version"
)

const (
	repoSlug   = "vobanai/voban-cli"
	assetPrefix = "voban-"
)

var (
	// githubAPI is the base URL of the GitHub REST API. Overridable in tests.
	githubAPI = "https://api.github.com"
	// executableFn resolves the path of the running binary. Overridable in tests.
	executableFn = os.Executable
	// httpClient is used for all upgrade HTTP calls.
	httpClient = &http.Client{Timeout: 5 * time.Minute}
)

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type ghRelease struct {
	TagName string    `json:"tag_name"`
	Assets  []ghAsset `json:"assets"`
}

// Run fetches the latest GitHub release and, if newer than the running build,
// downloads the matching platform asset, verifies its checksum, and atomically
// replaces the current executable. A "dev" build is always upgraded.
func Run(ctx context.Context) error {
	current := version.Version

	rel, err := latestRelease(ctx)
	if err != nil {
		return fmt.Errorf("fetch latest release: %w", err)
	}

	latest := strings.TrimPrefix(rel.TagName, "v")
	if !shouldUpgrade(current, latest) {
		fmt.Printf("voban is already up to date (%s)\n", latest)
		return nil
	}

	asset, err := pickAsset(rel.Assets, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return fmt.Errorf("select asset: %w", err)
	}

	bin, err := download(ctx, asset.BrowserDownloadURL)
	if err != nil {
		return fmt.Errorf("download %s: %w", asset.Name, err)
	}

	checks, err := download(ctx, checksumURL(rel.Assets))
	if err != nil {
		return fmt.Errorf("download checksums: %w", err)
	}
	if err := verifySHA256(bin, checks, asset.Name); err != nil {
		return fmt.Errorf("checksum %s: %w", asset.Name, err)
	}

	exePath, err := executableFn()
	if err != nil {
		return fmt.Errorf("resolve current executable: %w", err)
	}
	if err := replaceExecutable(exePath, bin); err != nil {
		return fmt.Errorf("replace %s: %w", exePath, err)
	}

	fmt.Printf("Upgraded voban %s → %s\n", displayVersion(current), latest)
	return nil
}

func latestRelease(ctx context.Context) (ghRelease, error) {
	var rel ghRelease
	url := fmt.Sprintf("%s/repos/%s/releases/latest", githubAPI, repoSlug)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return rel, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return rel, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return rel, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return rel, fmt.Errorf("github returned %s: %s", resp.Status, truncate(body, 200))
	}
	if err := json.Unmarshal(body, &rel); err != nil {
		return rel, fmt.Errorf("decode response: %w", err)
	}
	if rel.TagName == "" {
		return rel, errors.New("release has no tag_name")
	}
	return rel, nil
}

func pickAsset(assets []ghAsset, goos, goarch string) (ghAsset, error) {
	name := assetName(goos, goarch)
	for _, a := range assets {
		if a.Name == name {
			return a, nil
		}
	}
	available := make([]string, 0, len(assets))
	for _, a := range assets {
		available = append(available, a.Name)
	}
	return ghAsset{}, fmt.Errorf("no release asset for %s/%s (available: %s)",
		goos, goarch, strings.Join(available, ", "))
}

func assetName(goos, goarch string) string {
	if goos == "windows" {
		return assetPrefix + goos + "-" + goarch + ".exe"
	}
	return assetPrefix + goos + "-" + goarch
}

func checksumURL(assets []ghAsset) string {
	for _, a := range assets {
		if a.Name == "checksums.txt" {
			return a.BrowserDownloadURL
		}
	}
	return ""
}

func download(ctx context.Context, url string) ([]byte, error) {
	if url == "" {
		return nil, errors.New("empty URL")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s returned %s", url, resp.Status)
	}
	return body, nil
}

var checksumLine = regexp.MustCompile(`^([0-9a-fA-F]{64})\s+\*?(\S+)\s*$`)

func verifySHA256(body []byte, checksums []byte, assetName string) error {
	sum := sha256.Sum256(body)
	actual := hex.EncodeToString(sum[:])

	for _, line := range strings.Split(string(checksums), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := checksumLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if filepath.Base(m[2]) != assetName {
			continue
		}
		if !strings.EqualFold(m[1], actual) {
			return fmt.Errorf("mismatch: want %s, got %s", m[1], actual)
		}
		return nil
	}
	return fmt.Errorf("no checksum entry for %s in checksums.txt", assetName)
}

func replaceExecutable(exePath string, content []byte) error {
	dir := filepath.Dir(exePath)
	tmp, err := os.CreateTemp(dir, ".voban-update-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(tmpName, 0o755); err != nil {
			return fmt.Errorf("chmod temp file: %w", err)
		}
	}

	if runtime.GOOS == "windows" {
		old := exePath + ".old"
		_ = os.Remove(old)
		if err := os.Rename(exePath, old); err != nil {
			return fmt.Errorf("rename current executable: %w", err)
		}
		if err := os.Rename(tmpName, exePath); err != nil {
			_ = os.Rename(old, exePath)
			return fmt.Errorf("rename new executable: %w", err)
		}
		_ = os.Remove(old)
	} else {
		if err := os.Rename(tmpName, exePath); err != nil {
			return fmt.Errorf("rename: %w", err)
		}
	}
	cleanup = false
	return nil
}

// shouldUpgrade reports whether `latest` is newer than `current`. A "dev"
// build (or any non-semver current) always upgrades. Only simple X.Y.Z
// comparison is supported; pre-release suffixes on latest are treated as
// non-upgrades relative to the same X.Y.Z (rare for this repo).
func shouldUpgrade(current, latest string) bool {
	if latest == "" {
		return false
	}
	if !isSemver(current) {
		return true
	}
	return compareSemver(current, latest) < 0
}

var semverRe = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)`)

func isSemver(s string) bool {
	return semverRe.MatchString(s)
}

// compareSemver returns -1, 0, +1 comparing a and b. Both inputs may carry a
// leading "v"; only the leading X.Y.Z is compared, anything beyond is ignored.
func compareSemver(a, b string) int {
	av := parseSemver(a)
	bv := parseSemver(b)
	for i := 0; i < 3; i++ {
		if av[i] < bv[i] {
			return -1
		}
		if av[i] > bv[i] {
			return 1
		}
	}
	return 0
}

func parseSemver(s string) [3]int {
	var out [3]int
	m := semverRe.FindStringSubmatch(s)
	if m == nil {
		return out
	}
	for i := 0; i < 3; i++ {
		n, _ := strconv.Atoi(m[i+1])
		out[i] = n
	}
	return out
}

func displayVersion(v string) string {
	if v == "" {
		return "dev"
	}
	return v
}

func truncate(b []byte, n int) string {
	s := strings.TrimSpace(string(b))
	if len(s) > n {
		s = s[:n] + "..."
	}
	return s
}
