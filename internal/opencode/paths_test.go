package opencode

import (
	"path/filepath"
	"testing"
)

func TestConfigFileUsesXDGConfigHome(t *testing.T) {
	r := pathResolver{
		home: "/home/alice",
		env: map[string]string{
			"XDG_CONFIG_HOME": "/custom/config",
		},
	}
	want := filepath.Join("/custom/config", "opencode", "opencode.json")
	if got := r.configFile(); got != want {
		t.Errorf("configFile() = %q, want %q", got, want)
	}
}

func TestAuthFileUsesXDGDataHome(t *testing.T) {
	r := pathResolver{
		home: "/home/alice",
		env: map[string]string{
			"XDG_DATA_HOME": "/custom/data",
		},
	}
	want := filepath.Join("/custom/data", "opencode", "auth.json")
	if got := r.authFile(); got != want {
		t.Errorf("authFile() = %q, want %q", got, want)
	}
}

func TestConfigFileFallsBackToDotConfig(t *testing.T) {
	r := pathResolver{home: "/home/alice", env: map[string]string{}}
	want := filepath.Join("/home/alice", ".config", "opencode", "opencode.json")
	if got := r.configFile(); got != want {
		t.Errorf("configFile() = %q, want %q", got, want)
	}
}

func TestAuthFileFallsBackToLocalShare(t *testing.T) {
	r := pathResolver{home: "/home/alice", env: map[string]string{}}
	want := filepath.Join("/home/alice", ".local", "share", "opencode", "auth.json")
	if got := r.authFile(); got != want {
		t.Errorf("authFile() = %q, want %q", got, want)
	}
}

// opencode relies on the xdg-basedir npm package, which uses the same
// ~/.config and ~/.local/share segments on every platform, including macOS and
// Windows. It does NOT use ~/Library/Application Support or %APPDATA%. These
// tests pin that behavior using Windows-style and macOS-style home directories.
func TestConfigFileWindowsHomeUsesDotConfigSegments(t *testing.T) {
	r := pathResolver{home: `C:\Users\alice`, env: map[string]string{}}
	want := filepath.Join(`C:\Users\alice`, ".config", "opencode", "opencode.json")
	if got := r.configFile(); got != want {
		t.Errorf("configFile() = %q, want %q", got, want)
	}
}

func TestAuthFileMacHomeUsesLocalShareNotLibrary(t *testing.T) {
	r := pathResolver{home: "/Users/alice", env: map[string]string{}}
	want := filepath.Join("/Users/alice", ".local", "share", "opencode", "auth.json")
	if got := r.authFile(); got != want {
		t.Errorf("authFile() = %q, want %q", got, want)
	}
}

func TestEmptyXDGValueFallsBackToHome(t *testing.T) {
	r := pathResolver{
		home: "/home/alice",
		env:  map[string]string{"XDG_CONFIG_HOME": ""},
	}
	want := filepath.Join("/home/alice", ".config", "opencode", "opencode.json")
	if got := r.configFile(); got != want {
		t.Errorf("configFile() with empty XDG = %q, want %q", got, want)
	}
}
