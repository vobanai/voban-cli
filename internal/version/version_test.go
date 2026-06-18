package version

import (
	"runtime"
	"testing"
)

func TestInfo_ReleaseBuild(t *testing.T) {
	t.Cleanup(func() {
		Version = "dev"
		Commit = ""
	})

	Version = "1.2.3"
	Commit = "abc1234"

	v, c, goos, goarch := Info()
	if v != "1.2.3" {
		t.Errorf("version: got %q, want %q", v, "1.2.3")
	}
	if c != "abc1234" {
		t.Errorf("commit: got %q, want %q", c, "abc1234")
	}
	if goos != runtime.GOOS {
		t.Errorf("goos: got %q, want %q", goos, runtime.GOOS)
	}
	if goarch != runtime.GOARCH {
		t.Errorf("goarch: got %q, want %q", goarch, runtime.GOARCH)
	}
}

func TestInfo_DevFallbackUsesGit(t *testing.T) {
	t.Cleanup(restoreDefaults)

	Version = "dev"
	Commit = ""
	gitDescribeFn = func() (string, bool) { return "v0.1.0-3-g9a8b7c6-dirty", true }
	gitCommitFn = func() (string, bool) { return "9a8b7c6", true }

	v, c, _, _ := Info()
	if v != "v0.1.0-3-g9a8b7c6-dirty" {
		t.Errorf("version: got %q", v)
	}
	if c != "9a8b7c6" {
		t.Errorf("commit: got %q", c)
	}
}

func TestInfo_DevFallbackGitUnavailable(t *testing.T) {
	t.Cleanup(restoreDefaults)

	Version = "dev"
	Commit = ""
	gitDescribeFn = func() (string, bool) { return "", false }
	gitCommitFn = func() (string, bool) { return "", false }

	v, c, _, _ := Info()
	if v != "dev" {
		t.Errorf("version: got %q, want %q", v, "dev")
	}
	if c != "" {
		t.Errorf("commit: got %q, want empty", c)
	}
}

func restoreDefaults() {
	Version = "dev"
	Commit = ""
}
