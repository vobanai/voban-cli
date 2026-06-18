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
	c := client.New(baseURL, apiKey)

	metadata, err := c.ModelsMetadata(ctx)
	if err != nil {
		metadata = nil
	}
	if len(metadata) == 0 {
		ids, err := c.Models(ctx)
		if err != nil {
			return nil, fmt.Errorf("validate key and list models: %w", err)
		}
		if len(ids) == 0 {
			return nil, fmt.Errorf("no models available for this key; ask an admin to provision access")
		}
		metadata = metadataFromIDs(ids)
	}

	ids := make([]string, len(metadata))
	for i, m := range metadata {
		ids[i] = m.ID
	}

	resolver, err := newPathResolver()
	if err != nil {
		return nil, err
	}

	if err := writeOpenCodeConfig(resolver.configFile(), resolver.authFile(), baseURL+v1PathSuffix, apiKey, metadata); err != nil {
		return nil, err
	}
	return ids, nil
}

// metadataFromIDs builds minimal ModelMetadata entries from plain ID lists,
// used when the gateway does not expose /v1/models/metadata.
func metadataFromIDs(ids []string) []client.ModelMetadata {
	out := make([]client.ModelMetadata, len(ids))
	for i, id := range ids {
		out[i] = client.ModelMetadata{ID: id, Name: id}
	}
	return out
}

// writeOpenCodeConfig merges the Voban provider into opencode.json and the key
// into auth.json. Only the voban-owned entries are touched; all other user
// configuration is preserved.
func writeOpenCodeConfig(configPath, authPath, baseURL, apiKey string, models []client.ModelMetadata) error {
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

func modelEntries(models []client.ModelMetadata) map[string]any {
	entries := make(map[string]any, len(models))
	for _, m := range models {
		if strings.TrimSpace(m.ID) == "" {
			continue
		}
		entries[m.ID] = modelEntry(m)
	}
	return entries
}

func modelEntry(m client.ModelMetadata) map[string]any {
	entry := map[string]any{"name": m.Name}
	if m.Reasoning != nil {
		entry["reasoning"] = *m.Reasoning
	}
	if m.ToolCall != nil {
		entry["tool_call"] = *m.ToolCall
	}
	if m.Temperature != nil {
		entry["temperature"] = *m.Temperature
	}
	if m.Attachment != nil {
		entry["attachment"] = *m.Attachment
	}
	if m.Limit != nil {
		limit := map[string]any{"context": m.Limit.Context, "output": m.Limit.Output}
		if m.Limit.Input != nil {
			limit["input"] = *m.Limit.Input
		}
		entry["limit"] = limit
	}
	if m.Modalities != nil {
		modalities := map[string]any{}
		if len(m.Modalities.Input) > 0 {
			modalities["input"] = m.Modalities.Input
		}
		if len(m.Modalities.Output) > 0 {
			modalities["output"] = m.Modalities.Output
		}
		entry["modalities"] = modalities
	}
	variants := variantsFromReasoningOptions(m.ReasoningOptions)
	if len(variants) > 0 {
		entry["variants"] = variants
	}
	return entry
}

// variantsFromReasoningOptions builds opencode variant entries from models.dev
// reasoning_options. Only "effort" type produces variants; each value becomes a
// variant with body.reasoningEffort set, which @ai-sdk/openai-compatible
// normalizes into the OpenAI provider option.
func variantsFromReasoningOptions(options []client.ReasoningOption) map[string]any {
	variants := map[string]any{}
	for _, opt := range options {
		if opt.Type != "effort" {
			continue
		}
		for _, value := range opt.Values {
			if strings.TrimSpace(value) == "" {
				continue
			}
			variants[value] = map[string]any{
				"reasoningEffort": value,
			}
		}
	}
	return variants
}
