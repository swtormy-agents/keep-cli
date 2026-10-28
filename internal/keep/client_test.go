package keep

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientDoSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-KEY") != "test-key" {
			t.Fatalf("missing api key header")
		}
		if r.URL.Path != "/alerts" {
			t.Fatalf("path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"fingerprint":"abc"}]`))
	}))
	defer srv.Close()

	c := &Client{APIKey: "test-key", BaseURL: srv.URL, HTTP: srv.Client()}
	raw, err := c.Do(context.Background(), Request{Method: http.MethodGet, Path: "/alerts"})
	if err != nil {
		t.Fatal(err)
	}
	var arr []map[string]string
	if err := json.Unmarshal(raw, &arr); err != nil || arr[0]["fingerprint"] != "abc" {
		t.Fatalf("got %s", raw)
	}
}

func TestClientDoAuthError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"detail":"bad key"}`))
	}))
	defer srv.Close()

	c := &Client{APIKey: "bad", BaseURL: srv.URL, HTTP: srv.Client()}
	_, err := c.Do(context.Background(), Request{Method: http.MethodGet, Path: "/whoami"})
	var api *APIError
	if !errors.As(err, &api) || !api.IsAuth() {
		t.Fatalf("expected auth error, got %v", err)
	}
}
