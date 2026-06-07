package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunStatusReturnsSpendError(t *testing.T) {
	t.Setenv("VOBAN_API_KEY", "sk-sov-key")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/spend":
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":{"message":"spend unavailable"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	t.Setenv("VOBAN_BASE_URL", srv.URL)

	err := runStatus(t.Context())
	if err == nil {
		t.Fatal("expected runStatus to return spend error")
	}
	if got := err.Error(); !strings.Contains(got, "get spend") || !strings.Contains(got, "spend unavailable") {
		t.Errorf("error = %q, want spend failure context", got)
	}
}
