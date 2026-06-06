package opencode

import (
	"os"
	"path/filepath"
	"testing"
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

func TestConfigureWritesProviderBlock(t *testing.T) {
	configPath, authPath := writeConfigInputs(t)

	err := writeOpenCodeConfig(configPath, authPath, "https://api.voban.ai/v1", "sk-sov-k", []string{"glm-5.1", "devstral-2512"})
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

	if err := writeOpenCodeConfig(configPath, authPath, "https://api.voban.ai/v1", "sk-sov-k", []string{"glm-5.1", "devstral-2512"}); err != nil {
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

	if err := writeOpenCodeConfig(configPath, authPath, "https://api.voban.ai/v1", "sk-sov-k", []string{"glm-5.1"}); err != nil {
		t.Fatal(err)
	}

	if _, ok := readJSONFile(t, configPath)["model"]; ok {
		t.Error("config pinned a default model; the user should pick via /model_name")
	}
}

func TestConfigureWritesKeyToAuthNotConfig(t *testing.T) {
	configPath, authPath := writeConfigInputs(t)

	if err := writeOpenCodeConfig(configPath, authPath, "https://api.voban.ai/v1", "sk-sov-secret", []string{"glm-5.1"}); err != nil {
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

	if err := writeOpenCodeConfig(configPath, authPath, "https://api.voban.ai/v1", "sk-sov-k", []string{"glm-5.1"}); err != nil {
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

	if err := writeOpenCodeConfig(configPath, authPath, "https://api.voban.ai/v1", "sk-sov-k", []string{"glm-5.1"}); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeOpenCodeConfig(configPath, authPath, "https://api.voban.ai/v1", "sk-sov-k", []string{"glm-5.1"}); err != nil {
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

	if err := writeOpenCodeConfig(configPath, authPath, "https://api.voban.ai/v1", "sk-sov-k", []string{"old-model"}); err != nil {
		t.Fatal(err)
	}
	if err := writeOpenCodeConfig(configPath, authPath, "https://api.voban.ai/v1", "sk-sov-k", []string{"glm-5.1"}); err != nil {
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
