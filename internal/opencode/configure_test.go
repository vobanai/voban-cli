package opencode

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vobanai/voban-cli/internal/client"
)

func writeConfigInputs(t *testing.T) (configPath, authPath string) {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "config", "opencode.json"),
		filepath.Join(dir, "data", "auth.json")
}

func vobanProvider(t *testing.T, cfg map[string]any) map[string]any {
	t.Helper()
	provider, ok := cfg["provider"].(map[string]any)
	if !ok {
		t.Fatalf("provider missing: %v", cfg)
	}
	voban, ok := provider["voban"].(map[string]any)
	if !ok {
		t.Fatalf("voban provider missing: %v", provider)
	}
	return voban
}

func boolPtr(v bool) *bool { return &v }

func floatPtr(v float64) *float64 { return &v }

func simpleModels(ids ...string) []client.ModelMetadata {
	out := make([]client.ModelMetadata, len(ids))
	for i, id := range ids {
		out[i] = client.ModelMetadata{ID: id, Name: id}
	}
	return out
}

func TestConfigureWritesProviderBlock(t *testing.T) {
	configPath, authPath := writeConfigInputs(t)

	err := writeOpenCodeConfig(configPath, authPath, "https://api.voban.ai/v1", "sk-sov-k", simpleModels("glm-5.1", "devstral-2512"))
	if err != nil {
		t.Fatalf("writeOpenCodeConfig: %v", err)
	}

	cfg := readJSONFile(t, configPath)
	voban := vobanProvider(t, cfg)
	if voban["npm"] != "@ai-sdk/openai-compatible" {
		t.Errorf("npm = %v", voban["npm"])
	}
	if voban["name"] != "Voban" {
		t.Errorf("name = %v", voban["name"])
	}
	opts := voban["options"].(map[string]any)
	if opts["baseURL"] != "https://api.voban.ai/v1" {
		t.Errorf("baseURL = %v", opts["baseURL"])
	}
}

func TestConfigureRegistersDiscoveredModels(t *testing.T) {
	configPath, authPath := writeConfigInputs(t)

	if err := writeOpenCodeConfig(configPath, authPath, "https://api.voban.ai/v1", "sk-sov-k", simpleModels("glm-5.1", "devstral-2512")); err != nil {
		t.Fatal(err)
	}

	models := vobanProvider(t, readJSONFile(t, configPath))["models"].(map[string]any)
	for _, want := range []string{"glm-5.1", "devstral-2512"} {
		entry, ok := models[want].(map[string]any)
		if !ok {
			t.Fatalf("model %q missing", want)
		}
		if entry["name"] != want {
			t.Errorf("model %q name = %v", want, entry["name"])
		}
	}
}

func TestConfigureDoesNotPinDefaultModel(t *testing.T) {
	configPath, authPath := writeConfigInputs(t)

	if err := writeOpenCodeConfig(configPath, authPath, "https://api.voban.ai/v1", "sk-sov-k", simpleModels("glm-5.1")); err != nil {
		t.Fatal(err)
	}

	if _, ok := readJSONFile(t, configPath)["model"]; ok {
		t.Error("config pinned a default model; the user should pick via /models")
	}
}

func TestConfigureWritesKeyToAuthNotConfig(t *testing.T) {
	configPath, authPath := writeConfigInputs(t)

	if err := writeOpenCodeConfig(configPath, authPath, "https://api.voban.ai/v1", "sk-sov-secret", simpleModels("glm-5.1")); err != nil {
		t.Fatal(err)
	}

	opts := vobanProvider(t, readJSONFile(t, configPath))["options"].(map[string]any)
	if _, leaked := opts["apiKey"]; leaked {
		t.Error("API key leaked into opencode.json options")
	}
	auth := readJSONFile(t, authPath)
	if auth["voban"].(map[string]any)["key"] != "sk-sov-secret" {
		t.Error("API key not written to auth.json")
	}
}

func TestConfigurePreservesUnrelatedConfig(t *testing.T) {
	configPath, authPath := writeConfigInputs(t)
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	seed := `{"$schema":"custom-schema","theme":"tokyonight","provider":{"openrouter":{"name":"OpenRouter"}}}`
	if err := os.WriteFile(configPath, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := writeOpenCodeConfig(configPath, authPath, "https://api.voban.ai/v1", "sk-sov-k", simpleModels("glm-5.1")); err != nil {
		t.Fatal(err)
	}

	cfg := readJSONFile(t, configPath)
	if cfg["theme"] != "tokyonight" {
		t.Error("unrelated top-level key dropped")
	}
	if cfg["$schema"] != "custom-schema" {
		t.Error("unrelated $schema key changed")
	}
	provider := cfg["provider"].(map[string]any)
	if _, ok := provider["openrouter"]; !ok {
		t.Error("existing openrouter provider dropped")
	}
	if _, ok := provider["voban"]; !ok {
		t.Error("voban provider not added alongside openrouter")
	}
}

func TestConfigureIsIdempotent(t *testing.T) {
	configPath, authPath := writeConfigInputs(t)

	if err := writeOpenCodeConfig(configPath, authPath, "https://api.voban.ai/v1", "sk-sov-k", simpleModels("glm-5.1")); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeOpenCodeConfig(configPath, authPath, "https://api.voban.ai/v1", "sk-sov-k", simpleModels("glm-5.1")); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Error("running configure twice produced different config output")
	}
}

func TestConfigureReplacesStaleModelSet(t *testing.T) {
	configPath, authPath := writeConfigInputs(t)

	if err := writeOpenCodeConfig(configPath, authPath, "https://api.voban.ai/v1", "sk-sov-k", simpleModels("old-model")); err != nil {
		t.Fatal(err)
	}
	if err := writeOpenCodeConfig(configPath, authPath, "https://api.voban.ai/v1", "sk-sov-k", simpleModels("glm-5.1")); err != nil {
		t.Fatal(err)
	}

	models := vobanProvider(t, readJSONFile(t, configPath))["models"].(map[string]any)
	if _, ok := models["old-model"]; ok {
		t.Error("stale model retained after reconfigure")
	}
	if _, ok := models["glm-5.1"]; !ok {
		t.Error("new model missing after reconfigure")
	}
}

func TestConfigureWritesEnrichedMetadata(t *testing.T) {
	configPath, authPath := writeConfigInputs(t)

	models := []client.ModelMetadata{
		{
			ID:        "gpt-5.5",
			Name:      "GPT-5.5",
			Reasoning: boolPtr(true),
			ReasoningOptions: []client.ReasoningOption{
				{Type: "effort", Values: []string{"none", "low", "medium", "high", "xhigh"}},
			},
			Temperature: boolPtr(false),
			ToolCall:    boolPtr(true),
			Limit: &client.ModelLimit{
				Context: 1_050_000,
				Input:   floatPtr(922_000),
				Output:  128_000,
			},
		},
	}
	if err := writeOpenCodeConfig(configPath, authPath, "https://api.voban.ai/v1", "sk-sov-k", models); err != nil {
		t.Fatal(err)
	}

	entry := vobanProvider(t, readJSONFile(t, configPath))["models"].(map[string]any)["gpt-5.5"].(map[string]any)
	if entry["name"] != "GPT-5.5" {
		t.Fatalf("name = %v", entry["name"])
	}
	if entry["reasoning"] != true {
		t.Errorf("reasoning = %v, want true", entry["reasoning"])
	}
	if entry["temperature"] != false {
		t.Errorf("temperature = %v, want false", entry["temperature"])
	}
	if entry["tool_call"] != true {
		t.Errorf("tool_call = %v, want true", entry["tool_call"])
	}
	limit := entry["limit"].(map[string]any)
	if limit["context"] != float64(1_050_000) {
		t.Errorf("limit.context = %v", limit["context"])
	}
	if limit["output"] != float64(128_000) {
		t.Errorf("limit.output = %v", limit["output"])
	}

	variants := entry["variants"].(map[string]any)
	if len(variants) != 5 {
		t.Fatalf("expected 5 variants, got %d", len(variants))
	}
	high := variants["high"].(map[string]any)
	if high["reasoningEffort"] != "high" {
		t.Errorf("variant high reasoningEffort = %v", high["reasoningEffort"])
	}
}

func TestConfigureOmitsVariantsForNonReasoningModel(t *testing.T) {
	configPath, authPath := writeConfigInputs(t)

	models := []client.ModelMetadata{
		{ID: "devstral-2512", Name: "Devstral 2", Reasoning: boolPtr(false), ToolCall: boolPtr(true)},
	}
	if err := writeOpenCodeConfig(configPath, authPath, "https://api.voban.ai/v1", "sk-sov-k", models); err != nil {
		t.Fatal(err)
	}

	entry := vobanProvider(t, readJSONFile(t, configPath))["models"].(map[string]any)["devstral-2512"].(map[string]any)
	if _, hasVariants := entry["variants"]; hasVariants {
		t.Error("expected no variants for non-reasoning model")
	}
}

func TestConfigureOmitsVariantsForToggleOnlyReasoning(t *testing.T) {
	configPath, authPath := writeConfigInputs(t)

	models := []client.ModelMetadata{
		{
			ID:               "glm-5.1",
			Name:             "GLM-5.1",
			Reasoning:        boolPtr(true),
			ReasoningOptions: []client.ReasoningOption{{Type: "toggle"}},
			ToolCall:         boolPtr(true),
		},
	}
	if err := writeOpenCodeConfig(configPath, authPath, "https://api.voban.ai/v1", "sk-sov-k", models); err != nil {
		t.Fatal(err)
	}

	entry := vobanProvider(t, readJSONFile(t, configPath))["models"].(map[string]any)["glm-5.1"].(map[string]any)
	if _, hasVariants := entry["variants"]; hasVariants {
		t.Error("expected no variants for toggle-only reasoning model")
	}
}
