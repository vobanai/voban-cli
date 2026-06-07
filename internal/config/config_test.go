package config

import "testing"

func TestBaseURLDefault(t *testing.T) {
	t.Setenv("VOBAN_BASE_URL", "")
	if got := BaseURL(); got != "https://api.voban.ai" {
		t.Errorf("BaseURL() = %q, want https://api.voban.ai", got)
	}
}

func TestBaseURLOverride(t *testing.T) {
	t.Setenv("VOBAN_BASE_URL", "http://localhost:8080")
	if got := BaseURL(); got != "http://localhost:8080" {
		t.Errorf("BaseURL() = %q, want http://localhost:8080", got)
	}
}

func TestBaseURLOverrideStripsTrailingSlash(t *testing.T) {
	t.Setenv("VOBAN_BASE_URL", "http://localhost:8080/")
	if got := BaseURL(); got != "http://localhost:8080" {
		t.Errorf("BaseURL() = %q, want trailing slash stripped", got)
	}
}

func TestValidateAPIKeyAcceptsSovPrefix(t *testing.T) {
	if err := ValidateAPIKey("sk-sov-abc123"); err != nil {
		t.Errorf("ValidateAPIKey returned error for valid key: %v", err)
	}
}

func TestValidateAPIKeyRejectsWrongPrefix(t *testing.T) {
	if err := ValidateAPIKey("sk-svc-abc123"); err == nil {
		t.Error("ValidateAPIKey accepted a non sk-sov- key")
	}
}

func TestValidateAPIKeyRejectsEmpty(t *testing.T) {
	if err := ValidateAPIKey(""); err == nil {
		t.Error("ValidateAPIKey accepted an empty key")
	}
}

func TestValidateAPIKeyRejectsPrefixOnly(t *testing.T) {
	if err := ValidateAPIKey("sk-sov-"); err == nil {
		t.Error("ValidateAPIKey accepted the prefix with no random part")
	}
}
