package upgrade

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/vobanai/voban-cli/internal/version"
)

// newFakeServer serves the GitHub release JSON, the binary asset and the
// checksums file from an httptest server. The returned string is the server
// base URL to assign to githubAPI.
func newFakeServer(t *testing.T, tag string, assets map[string][]byte) string {
	t.Helper()
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/repos/vobanai/voban-cli/releases/latest":
			rel := ghRelease{TagName: tag}
			for name, body := range assets {
				rel.Assets = append(rel.Assets, ghAsset{
					Name:               name,
					BrowserDownloadURL: srv.URL + "/download/" + name,
					Size:               int64(len(body)),
				})
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(rel)
		case strings.HasPrefix(r.URL.Path, "/download/"):
			name := strings.TrimPrefix(r.URL.Path, "/download/")
			body, ok := assets[name]
			if !ok {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write(body)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func newFakeExe(t *testing.T) (exePath, dir string) {
	t.Helper()
	dir = t.TempDir()
	exePath = filepath.Join(dir, exeName("voban"))
	if err := os.WriteFile(exePath, []byte("old-bytes"), 0o755); err != nil {
		t.Fatalf("write fake exe: %v", err)
	}
	return exePath, dir
}

func exeName(base string) string {
	if runtime.GOOS == "windows" {
		return base + ".exe"
	}
	return base
}

func checksums(assets map[string][]byte) []byte {
	var b strings.Builder
	for name, body := range assets {
		if name == "checksums.txt" {
			continue
		}
		sum := sha256.Sum256(body)
		b.WriteString(hex.EncodeToString(sum[:]))
		b.WriteString("  ")
		b.WriteString(name)
		b.WriteString("\n")
	}
	return []byte(b.String())
}

func platformAssetName() string {
	return assetName(runtime.GOOS, runtime.GOARCH)
}

// setup overrides the package-level hooks for the duration of the test and
// points os.Executable at the fake binary path.
func setup(t *testing.T, exePath string) {
	t.Helper()
	prevAPI, prevExe, prevHTTP := githubAPI, executableFn, httpClient
	t.Cleanup(func() {
		githubAPI = prevAPI
		executableFn = prevExe
		httpClient = prevHTTP
	})
	httpClient = &http.Client{}
	executableFn = func() (string, error) { return exePath, nil }
}

func TestRun_AlreadyUpToDate(t *testing.T) {
	version.Version = "0.0.6"
	t.Cleanup(func() { version.Version = "dev" })

	assets := map[string][]byte{
		platformAssetName(): []byte("new-bytes"),
		"checksums.txt":     checksums(map[string][]byte{platformAssetName(): []byte("new-bytes")}),
	}
	githubAPI = newFakeServer(t, "v0.0.6", assets)

	exePath, _ := newFakeExe(t)
	setup(t, exePath)

	if err := Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	got, err := os.ReadFile(exePath)
	if err != nil {
		t.Fatalf("read exe: %v", err)
	}
	if string(got) != "old-bytes" {
		t.Fatalf("executable was modified despite being up to date: %q", got)
	}
}

func TestRun_UpgradesToNewer(t *testing.T) {
	version.Version = "0.0.5"
	t.Cleanup(func() { version.Version = "dev" })

	assetBody := []byte("brand-new-binary")
	assets := map[string][]byte{
		platformAssetName(): assetBody,
		"checksums.txt":     checksums(map[string][]byte{platformAssetName(): assetBody}),
	}
	githubAPI = newFakeServer(t, "v0.0.6", assets)

	exePath, _ := newFakeExe(t)
	setup(t, exePath)

	if err := Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	got, err := os.ReadFile(exePath)
	if err != nil {
		t.Fatalf("read exe: %v", err)
	}
	if string(got) != "brand-new-binary" {
		t.Fatalf("exe content = %q, want %q", got, "brand-new-binary")
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(exePath)
		if err != nil {
			t.Fatalf("stat exe: %v", err)
		}
		if info.Mode().Perm() != 0o755 {
			t.Fatalf("exe perms = %v, want 0o755", info.Mode().Perm())
		}
	}
}

func TestRun_DevBuildUpgrades(t *testing.T) {
	version.Version = "dev"
	t.Cleanup(func() { version.Version = "dev" })

	assetBody := []byte("from-dev")
	assets := map[string][]byte{
		platformAssetName(): assetBody,
		"checksums.txt":     checksums(map[string][]byte{platformAssetName(): assetBody}),
	}
	githubAPI = newFakeServer(t, "v0.0.6", assets)

	exePath, _ := newFakeExe(t)
	setup(t, exePath)

	if err := Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := os.ReadFile(exePath)
	if string(got) != "from-dev" {
		t.Fatalf("exe content = %q, want %q", got, "from-dev")
	}
}

func TestRun_MissingPlatformAsset(t *testing.T) {
	version.Version = "0.0.5"
	t.Cleanup(func() { version.Version = "dev" })

	otherName := "voban-plan9-386"
	assets := map[string][]byte{
		otherName:       []byte("nope"),
		"checksums.txt": checksums(map[string][]byte{otherName: []byte("nope")}),
	}
	githubAPI = newFakeServer(t, "v0.0.6", assets)

	exePath, _ := newFakeExe(t)
	setup(t, exePath)

	err := Run(context.Background())
	if err == nil {
		t.Fatal("expected error for missing platform asset")
	}
	if !strings.Contains(err.Error(), "no release asset") {
		t.Fatalf("error = %q, want it to mention missing asset", err)
	}

	got, _ := os.ReadFile(exePath)
	if string(got) != "old-bytes" {
		t.Fatalf("exe should be untouched, got %q", got)
	}
}

func TestRun_ChecksumMismatch(t *testing.T) {
	version.Version = "0.0.5"
	t.Cleanup(func() { version.Version = "dev" })

	assetBody := []byte("good-bin")
	assets := map[string][]byte{
		platformAssetName(): assetBody,
		"checksums.txt":     []byte("0000000000000000000000000000000000000000000000000000000000000000  " + platformAssetName() + "\n"),
	}
	githubAPI = newFakeServer(t, "v0.0.6", assets)

	exePath, _ := newFakeExe(t)
	setup(t, exePath)

	err := Run(context.Background())
	if err == nil {
		t.Fatal("expected checksum error")
	}
	if !strings.Contains(err.Error(), "mismatch") {
		t.Fatalf("error = %q, want checksum mismatch", err)
	}

	got, _ := os.ReadFile(exePath)
	if string(got) != "old-bytes" {
		t.Fatalf("exe should be untouched on checksum failure, got %q", got)
	}
}

func TestRun_MissingChecksumEntry(t *testing.T) {
	version.Version = "0.0.5"
	t.Cleanup(func() { version.Version = "dev" })

	assetBody := []byte("good-bin")
	assets := map[string][]byte{
		platformAssetName(): assetBody,
		"checksums.txt":     []byte("deadbeef  other-asset\n"),
	}
	githubAPI = newFakeServer(t, "v0.0.6", assets)

	exePath, _ := newFakeExe(t)
	setup(t, exePath)

	err := Run(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no checksum entry") {
		t.Fatalf("error = %q, want no checksum entry", err)
	}
}

func TestRun_GitHubAPIError(t *testing.T) {
	version.Version = "0.0.5"
	t.Cleanup(func() { version.Version = "dev" })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, "boom")
	}))
	t.Cleanup(srv.Close)
	githubAPI = srv.URL

	exePath, _ := newFakeExe(t)
	setup(t, exePath)

	err := Run(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "fetch latest release") {
		t.Fatalf("error = %q, want fetch latest release wrap", err)
	}
}

func TestShouldUpgrade(t *testing.T) {
	cases := []struct {
		current, latest string
		want            bool
	}{
		{"0.0.5", "0.0.6", true},
		{"0.0.6", "0.0.6", false},
		{"0.1.0", "0.0.9", false},
		{"1.0.0", "0.9.9", false},
		{"dev", "0.0.6", true},
		{"", "0.0.6", true},
		{"v0.0.5", "0.0.6", true},
		{"0.0.5-rc1", "0.0.5", false},
		{"0.0.6", "", false},
	}
	for _, c := range cases {
		if got := shouldUpgrade(c.current, c.latest); got != c.want {
			t.Errorf("shouldUpgrade(%q, %q) = %v, want %v", c.current, c.latest, got, c.want)
		}
	}
}

func TestCompareSemver(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"0.0.5", "0.0.6", -1},
		{"0.0.6", "0.0.6", 0},
		{"0.0.7", "0.0.6", 1},
		{"v0.0.5", "0.0.6", -1},
		{"1.2.3", "1.2.10", -1},
		{"1.10.0", "1.2.0", 1},
	}
	for _, c := range cases {
		if got := compareSemver(c.a, c.b); got != c.want {
			t.Errorf("compareSemver(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestPickAsset(t *testing.T) {
	assets := []ghAsset{
		{Name: "voban-linux-amd64"},
		{Name: "voban-darwin-arm64"},
		{Name: "voban-windows-amd64.exe"},
	}
	got, err := pickAsset(assets, "darwin", "arm64")
	if err != nil {
		t.Fatalf("pickAsset: %v", err)
	}
	if got.Name != "voban-darwin-arm64" {
		t.Errorf("asset = %q, want voban-darwin-arm64", got.Name)
	}

	if _, err := pickAsset(assets, "plan9", "386"); err == nil {
		t.Fatal("expected error for unknown platform")
	}
}
