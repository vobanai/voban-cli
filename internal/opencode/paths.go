package opencode

import (
	"fmt"
	"os"
	"path/filepath"
)

const appDir = "opencode"

// pathResolver locates opencode's global config and data files. It replicates
// the resolution used by opencode's xdg-basedir dependency rather than Go's
// os.UserConfigDir, because opencode uses ~/.config and ~/.local/share on every
// platform (macOS and Windows included), not the native OS directories.
type pathResolver struct {
	home string
	env  map[string]string
}

func newPathResolver() (pathResolver, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return pathResolver{}, fmt.Errorf("resolve home directory: %w", err)
	}
	return pathResolver{
		home: home,
		env: map[string]string{
			"XDG_CONFIG_HOME": os.Getenv("XDG_CONFIG_HOME"),
			"XDG_DATA_HOME":   os.Getenv("XDG_DATA_HOME"),
		},
	}, nil
}

func (r pathResolver) configDir() string {
	if v := r.env["XDG_CONFIG_HOME"]; v != "" {
		return filepath.Join(v, appDir)
	}
	return filepath.Join(r.home, ".config", appDir)
}

func (r pathResolver) dataDir() string {
	if v := r.env["XDG_DATA_HOME"]; v != "" {
		return filepath.Join(v, appDir)
	}
	return filepath.Join(r.home, ".local", "share", appDir)
}

func (r pathResolver) configFile() string {
	return filepath.Join(r.configDir(), "opencode.json")
}

func (r pathResolver) authFile() string {
	return filepath.Join(r.dataDir(), "auth.json")
}
