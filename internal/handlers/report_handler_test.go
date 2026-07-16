package handlers

import (
	"errors"
	"net/http"
	"testing"
)

func TestNormalizeGeminiProxyErrorDoesNotExposeProviderAuthStatus(t *testing.T) {
	for _, providerStatus := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		status, response := normalizeGeminiProxyError(providerStatus, errors.New("provider rejected credentials"))
		if status != http.StatusBadGateway {
			t.Fatalf("provider status %d mapped to %d, want %d", providerStatus, status, http.StatusBadGateway)
		}
		if response.Error != "GEMINI_AUTH_ERROR" {
			t.Fatalf("error code = %q, want GEMINI_AUTH_ERROR", response.Error)
		}
	}
}

func TestNormalizeGeminiProxyErrorPreservesNonAuthStatus(t *testing.T) {
	status, response := normalizeGeminiProxyError(http.StatusTooManyRequests, errors.New("quota exceeded"))
	if status != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", status, http.StatusTooManyRequests)
	}
	if response.Error != "GEMINI_ERROR" || response.Message != "quota exceeded" {
		t.Fatalf("response = %#v", response)
	}
}
