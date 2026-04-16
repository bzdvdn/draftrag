package draftrag

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestWeaviate_RedactsAPIKeyFromErrorBody(t *testing.T) {
	apiKey := "secret-key"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("unauthorized: " + apiKey))
	}))
	t.Cleanup(srv.Close)

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}

	opts := WeaviateOptions{
		Scheme:     u.Scheme,
		Host:       u.Host,
		Collection: "test",
		APIKey:     apiKey,
	}

	err = CreateWeaviateCollection(context.Background(), opts)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if strings.Contains(err.Error(), apiKey) {
		t.Fatalf("expected APIKey to be redacted from error, got: %v", err)
	}
}
