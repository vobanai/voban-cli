package opencode

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoredKeyReadsVobanEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	if err := writeVobanAuth(path, "sk-sov-stored"); err != nil {
		t.Fatal(err)
	}

	key, ok, err := storedKey(path)
	if err != nil {
		t.Fatalf("storedKey returned error: %v", err)
	}
	if !ok || key != "sk-sov-stored" {
		t.Errorf("storedKey() = %q, %v; want sk-sov-stored, true", key, ok)
	}
}

func TestStoredKeyMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	_, ok, err := storedKey(path)
	if err != nil {
		t.Fatalf("storedKey returned error: %v", err)
	}
	if ok {
		t.Error("storedKey reported a key for a missing file")
	}
}

func TestStoredKeyNoVobanEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(path, []byte(`{"openrouter":{"type":"api","key":"x"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, ok, err := storedKey(path)
	if err != nil {
		t.Fatalf("storedKey returned error: %v", err)
	}
	if ok {
		t.Error("storedKey reported a key when no voban entry exists")
	}
}

func TestStoredKeyReturnsReadError(t *testing.T) {
	_, ok, err := storedKey(t.TempDir())
	if err == nil {
		t.Fatal("expected error when auth path cannot be read as a file")
	}
	if ok {
		t.Error("storedKey reported a key when reading auth failed")
	}
}
