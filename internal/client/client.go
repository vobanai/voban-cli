// Package client is a minimal HTTP client for the Voban gateway endpoints the
// CLI needs: identity, spend, and the OpenAI-compatible model list.
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client talks to a Voban gateway using a user API key.
type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// New builds a Client for the given gateway base URL and sk-sov- API key.
func New(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// Spend is the budget window and blocked state from GET /v1/spend.
type Spend struct {
	UserID         string  `json:"user_id"`
	Spend          float64 `json:"spend"`
	MaxBudget      float64 `json:"max_budget"`
	BudgetDuration string  `json:"budget_duration"`
	BudgetResetAt  string  `json:"budget_reset_at"`
	Blocked        bool    `json:"blocked"`
}

func (c *Client) Spend(ctx context.Context) (Spend, error) {
	var spend Spend
	if err := c.getJSON(ctx, "/v1/spend", &spend); err != nil {
		return Spend{}, err
	}
	return spend, nil
}

// Models returns the model IDs available to the key from GET /v1/models,
// already filtered by the gateway for the caller's organization.
func (c *Client) Models(ctx context.Context) ([]string, error) {
	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := c.getJSON(ctx, "/v1/models", &payload); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(payload.Data))
	for _, m := range payload.Data {
		if m.ID != "" {
			ids = append(ids, m.ID)
		}
	}
	return ids, nil
}

func (c *Client) getJSON(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("build request for %s: %w", path, err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request %s: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response from %s: %w", path, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s returned %s: %s", path, resp.Status, errorMessage(body))
	}

	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode response from %s: %w", path, err)
	}
	return nil
}

// errorMessage extracts the OpenAI-style {"error":{"message":...}} text the
// gateway returns, falling back to the raw body when it is not that shape.
func errorMessage(body []byte) string {
	var envelope struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil && envelope.Error.Message != "" {
		return envelope.Error.Message
	}
	return string(body)
}
