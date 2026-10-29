package keep

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestClientDoRedactsAPIErrorMessage verifies the HTTP client redacts API error
// bodies before they surface on APIError.Message (without going through CLI reportError).
func TestClientDoRedactsAPIErrorMessage(t *testing.T) {
	const secret = "client-layer-secret-key"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"detail":"bad key ` + secret + `"}`))
	}))
	defer srv.Close()

	c := &Client{APIKey: secret, BaseURL: srv.URL, HTTP: srv.Client()}
	_, err := c.Do(context.Background(), Request{Method: http.MethodGet, Path: "/whoami"})
	if err == nil {
		t.Fatal("expected error")
	}
	api, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if strings.Contains(api.Message, secret) {
		t.Fatalf("client.Do did not redact APIError.Message: %q", api.Message)
	}
}
