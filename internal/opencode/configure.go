package opencode

import (
	"context"
	"fmt"
	"strings"

	"github.com/vobanai/voban-cli/internal/client"
	"github.com/vobanai/voban-cli/internal/config"
)

const (
	providerNPM  = "@ai-sdk/openai-compatible"
	providerName = "Voban"
	v1PathSuffix = "/v1"
)

// Configure validates the API key against the gateway, discovers the available
// models, and writes opencode's config and auth files. It returns the model IDs
// that were registered so the caller can report them.
func Configure(ctx context.Context, apiKey string) ([]string, error) {
	if err := config.ValidateAPIKey(apiKey); err != nil {
		return nil, err
	}

	baseURL := config.BaseURL()
	models, err := client.New(baseURL, apiKey).Models(ctx)
	if err != nil {
		return nil, fmt.Errorf("validate key and list models: %w", err)
	}
	if len(models) == 0 {
		return nil, fmt.Errorf("no models available for this key; ask an admin to provision access")
	}

	resolver, err := newPathResolver()
	if err != nil {
		return nil, err
	}

	if err := writeOpenCodeConfig(resolver.configFile(), resolver.authFile(), baseURL+v1PathSuffix, apiKey, models); err != nil {
		return nil, err
	}
	return models, nil
}

// writeOpenCodeConfig merges the Voban provider into opencode.json and the key
// into auth.json. Only the voban-owned entries are touched; all other user
// configuration is preserved.
func writeOpenCodeConfig(configPath, authPath, baseURL, apiKey string, models []string) error {
	cfg, err := readJSONObject(configPath)
	if err != nil {
		return err
	}

	provider, ok := cfg["provider"].(map[string]any)
	if !ok {
		provider = map[string]any{}
	}
	provider[providerID] = map[string]any{
		"npm":     providerNPM,
		"name":    providerName,
		"options": map[string]any{"baseURL": baseURL},
		"models":  modelEntries(models),
	}
	cfg["provider"] = provider

	if err := writeJSONObject(configPath, cfg, 0o644); err != nil {
		return err
	}
	return writeVobanAuth(authPath, apiKey)
}

func modelEntries(models []string) map[string]any {
	entries := make(map[string]any, len(models))
	for _, id := range models {
		if strings.TrimSpace(id) == "" {
			continue
		}
		entries[id] = map[string]any{"name": id}
	}
	return entries
}
