// Package version exposes the CLI build version and commit, resolved either
// from ldflag-injected values (release builds) or from git at runtime (dev builds).
package version

import (
	"os/exec"
	"runtime"
	"strings"
)

var (
	Version = "dev" // overridden via -ldflags -X at release build time
	Commit  = ""    // overridden via -ldflags -X at release build time
)

// gitDescribeFn returns a git version string (e.g. "v0.3.0-2-g1a2b3c4-dirty").
// Returns ok=false if git is unavailable or the cwd is not a repository.
var gitDescribeFn = func() (string, bool) { return gitOutput("describe", "--tags", "--always", "--dirty") }

// gitCommitFn returns the short commit hash.
var gitCommitFn = func() (string, bool) { return gitOutput("rev-parse", "--short", "HEAD") }

// Info resolves the version, commit and platform for display. When the
// ldflag-injected Version is a real release (anything other than "dev"), it is
// returned as-is. Otherwise, it falls back to `git describe` at runtime. The
// commit follows the same rule: ldflag value first, then git, then empty.
func Info() (version, commit, goos, goarch string) {
	version = Version
	if version == "dev" {
		if v, ok := gitDescribeFn(); ok {
			version = v
		}
	}

	commit = Commit
	if commit == "" {
		if c, ok := gitCommitFn(); ok {
			commit = c
		}
	}

	return version, commit, runtime.GOOS, runtime.GOARCH
}

func gitOutput(args ...string) (string, bool) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return "", false
	}
	return s, true
}
