package opencode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// readJSONObject loads a JSON object from path. A missing or unparseable file is
// treated as an empty object so a single corrupt file never blocks configuration.
func readJSONObject(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil || out == nil {
		return map[string]any{}, nil
	}
	return out, nil
}

// writeJSONObject writes obj as indented JSON to path, creating parent
// directories as needed and applying perm to the file.
func writeJSONObject(path string, obj map[string]any, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create directory for %s: %w", path, err)
	}
	data, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, perm); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := os.Chmod(path, perm); err != nil {
		return fmt.Errorf("set permissions on %s: %w", path, err)
	}
	return nil
}
