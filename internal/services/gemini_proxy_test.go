package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGeminiProxyGenerateContent(t *testing.T) {
	var received geminiGenerateRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1beta/models/test-model:generateContent" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("x-goog-api-key"); got != "test-key" {
			t.Errorf("x-goog-api-key = %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Errorf("decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"ok"}]},"finishReason":"STOP"}]}`))
	}))
	defer server.Close()

	service := NewGeminiProxyService(GeminiProxyConfig{
		APIKey:          " test-key ",
		Model:           " test-model ",
		APIBaseURL:      server.URL + "/v1beta/",
		EnableWebSearch: true,
		MaxOutputTokens: 512,
	})

	response, status, err := service.GenerateContent(GeminiProxyRequest{
		Contents: []GeminiContent{{Role: "user", Parts: []GeminiTextPart{{Text: "compare"}}}},
	})
	if err != nil {
		t.Fatalf("GenerateContent() error = %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if got := response.Candidates[0].Content.Parts[0].Text; got != "ok" {
		t.Fatalf("response text = %q", got)
	}
	if received.GenerationConfig.MaxOutputTokens != 512 {
		t.Fatalf("maxOutputTokens = %d", received.GenerationConfig.MaxOutputTokens)
	}
	if len(received.Tools) != 1 {
		t.Fatalf("tools count = %d, want 1", len(received.Tools))
	}
}

func TestGeminiProxyGenerateContentReturnsProviderError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"message":"key rejected"}}`))
	}))
	defer server.Close()

	service := NewGeminiProxyService(GeminiProxyConfig{
		APIKey:     "test-key",
		Model:      "test-model",
		APIBaseURL: server.URL,
	})

	response, status, err := service.GenerateContent(GeminiProxyRequest{})
	if response != nil {
		t.Fatal("expected nil response")
	}
	if status != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", status)
	}
	if err == nil || !strings.Contains(err.Error(), "key rejected") {
		t.Fatalf("error = %v", err)
	}
}

func TestGeminiProxyConfiguration(t *testing.T) {
	service := NewGeminiProxyService(GeminiProxyConfig{})
	if service.IsConfigured() {
		t.Fatal("empty config must not be configured")
	}
	if service.apiBaseURL != defaultGeminiAPIBaseURL {
		t.Fatalf("apiBaseURL = %q", service.apiBaseURL)
	}
}
