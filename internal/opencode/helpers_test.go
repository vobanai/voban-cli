package opencode

import (
	"encoding/json"
	"os"
	"testing"
)

// readJSONFile loads a JSON object from path and fails the test if it is missing
// or not valid JSON. Shared by the auth and configure tests.
func readJSONFile(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("%s is not valid JSON: %v", path, err)
	}
	return out
}
