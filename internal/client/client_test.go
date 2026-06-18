package client

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSpendReturnsBudget(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer sk-sov-key" {
			t.Errorf("Authorization = %q, want Bearer sk-sov-key", got)
		}
		if r.URL.Path != "/v1/spend" {
			t.Errorf("path = %q, want /v1/spend", r.URL.Path)
		}
		w.Write([]byte(`{"user_id":"u1","spend":1.5,"max_budget":10,"blocked":false}`))
	}))
	defer srv.Close()

	spend, err := New(srv.URL, "sk-sov-key").Spend(t.Context())
	if err != nil {
		t.Fatalf("Spend() error: %v", err)
	}
	if spend.UserID != "u1" || spend.Spend != 1.5 || spend.MaxBudget != 10 || spend.Blocked {
		t.Errorf("unexpected Spend result: %+v", spend)
	}
}

func TestModelsReturnsIDs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("path = %q, want /v1/models", r.URL.Path)
		}
		w.Write([]byte(`{"object":"list","data":[{"id":"glm-5.1"},{"id":"devstral-2512"}]}`))
	}))
	defer srv.Close()

	models, err := New(srv.URL, "sk-sov-key").Models(t.Context())
	if err != nil {
		t.Fatalf("Models() error: %v", err)
	}
	if len(models) != 2 || models[0] != "glm-5.1" || models[1] != "devstral-2512" {
		t.Errorf("unexpected models: %v", models)
	}
}

func TestModelsParsesOpenAIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":{"type":"insufficient_quota","message":"no budget"}}`))
	}))
	defer srv.Close()

	_, err := New(srv.URL, "sk-sov-key").Models(t.Context())
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
	if got := err.Error(); !strings.Contains(got, "no budget") {
		t.Errorf("error = %q, want it to include the upstream message", got)
	}
}

func TestModelsMetadataReturnsEnrichedModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models/metadata" {
			t.Errorf("path = %q, want /v1/models/metadata", r.URL.Path)
		}
		w.Write([]byte(`{"object":"list","data":[{"id":"gpt-5.5","metadata_id":"openai/gpt-5.5","name":"GPT-5.5","reasoning":true,"reasoning_options":[{"type":"effort","values":["low","high"]}],"tool_call":true,"temperature":false,"limit":{"context":1050000,"output":128000}}]}`))
	}))
	defer srv.Close()

	models, err := New(srv.URL, "sk-sov-key").ModelsMetadata(t.Context())
	if err != nil {
		t.Fatalf("ModelsMetadata() error: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}
	m := models[0]
	if m.ID != "gpt-5.5" || m.Name != "GPT-5.5" {
		t.Fatalf("unexpected model: %+v", m)
	}
	if m.Reasoning == nil || !*m.Reasoning {
		t.Fatalf("expected reasoning=true, got %v", m.Reasoning)
	}
	if len(m.ReasoningOptions) != 1 || m.ReasoningOptions[0].Type != "effort" {
		t.Fatalf("unexpected reasoning options: %+v", m.ReasoningOptions)
	}
	if len(m.ReasoningOptions[0].Values) != 2 || m.ReasoningOptions[0].Values[0] != "low" || m.ReasoningOptions[0].Values[1] != "high" {
		t.Fatalf("unexpected effort values: %v", m.ReasoningOptions[0].Values)
	}
}

func TestUnauthorizedSurfacesStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"type":"unauthorized","message":"invalid key"}}`))
	}))
	defer srv.Close()

	_, err := New(srv.URL, "sk-sov-key").Spend(t.Context())
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}
