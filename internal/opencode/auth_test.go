package opencode

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWriteAuthCreatesFileWithKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "auth.json")

	if err := writeVobanAuth(path, "sk-sov-abc"); err != nil {
		t.Fatalf("writeVobanAuth: %v", err)
	}

	entries := readJSONFile(t, path)
	voban, ok := entries["voban"].(map[string]any)
	if !ok {
		t.Fatalf("voban entry missing or wrong type: %v", entries["voban"])
	}
	if voban["type"] != "api" || voban["key"] != "sk-sov-abc" {
		t.Errorf("unexpected voban auth entry: %v", voban)
	}
}

func TestWriteAuthPreservesOtherProviders(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	seed := `{"openrouter":{"type":"api","key":"sk-or-keep"}}`
	if err := os.WriteFile(path, []byte(seed), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := writeVobanAuth(path, "sk-sov-new"); err != nil {
		t.Fatalf("writeVobanAuth: %v", err)
	}

	entries := readJSONFile(t, path)
	if _, ok := entries["openrouter"]; !ok {
		t.Error("existing openrouter entry was dropped")
	}
	voban := entries["voban"].(map[string]any)
	if voban["key"] != "sk-sov-new" {
		t.Errorf("voban key = %v, want sk-sov-new", voban["key"])
	}
}

func TestWriteAuthOverwritesExistingVobanKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	if err := writeVobanAuth(path, "sk-sov-old"); err != nil {
		t.Fatal(err)
	}
	if err := writeVobanAuth(path, "sk-sov-rotated"); err != nil {
		t.Fatal(err)
	}

	entries := readJSONFile(t, path)
	voban := entries["voban"].(map[string]any)
	if voban["key"] != "sk-sov-rotated" {
		t.Errorf("voban key = %v, want sk-sov-rotated", voban["key"])
	}
}

func TestWriteAuthSetsRestrictivePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix file permissions not enforced on Windows")
	}
	path := filepath.Join(t.TempDir(), "auth.json")
	if err := writeVobanAuth(path, "sk-sov-abc"); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("auth.json perm = %o, want 600", perm)
	}
}

func TestWriteAuthTightensExistingPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix file permissions not enforced on Windows")
	}
	path := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(path, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := writeVobanAuth(path, "sk-sov-abc"); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("auth.json perm = %o, want 600", perm)
	}
}

func TestWriteAuthTreatsCorruptFileAsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := writeVobanAuth(path, "sk-sov-abc"); err != nil {
		t.Fatalf("writeVobanAuth should recover from corrupt file: %v", err)
	}
	entries := readJSONFile(t, path)
	if entries["voban"].(map[string]any)["key"] != "sk-sov-abc" {
		t.Error("voban entry not written over corrupt file")
	}
}
