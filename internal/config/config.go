// Package config holds the gateway base URL resolution and API key validation
// shared across the voban CLI commands.
package config

import (
	"fmt"
	"os"
	"strings"
)

const (
	defaultBaseURL = "https://api.voban.ai"
	apiKeyPrefix   = "sk-sov-"
)

// BaseURL returns the Voban gateway base URL, honoring the VOBAN_BASE_URL
// override for self-hosted or development instances. Any trailing slash is
// stripped so callers can append paths without doubling separators.
func BaseURL() string {
	if override := strings.TrimSpace(os.Getenv("VOBAN_BASE_URL")); override != "" {
		return strings.TrimRight(override, "/")
	}
	return defaultBaseURL
}

// ValidateAPIKey checks that key is a well-formed Voban user API key. It must
// start with the sk-sov- prefix and carry a non-empty random part. This mirrors
// the gateway's own prefix check so users get a clear error before any request.
func ValidateAPIKey(key string) error {
	if key == "" {
		return fmt.Errorf("API key is empty")
	}
	if !strings.HasPrefix(key, apiKeyPrefix) {
		return fmt.Errorf("API key must start with %q", apiKeyPrefix)
	}
	if strings.TrimPrefix(key, apiKeyPrefix) == "" {
		return fmt.Errorf("API key is missing its random part after %q", apiKeyPrefix)
	}
	return nil
}
