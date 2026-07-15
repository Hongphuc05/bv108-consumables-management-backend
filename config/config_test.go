package config

import "testing"

func TestGetEnv(t *testing.T) {
	const key = "CODEX_TEST_ENV_VALUE"

	t.Run("returns default when unset", func(t *testing.T) {
		t.Setenv(key, "")
		if value := getEnv(key+"__UNSET", "fallback"); value != "fallback" {
			t.Fatalf("expected fallback for unset env, got %q", value)
		}
	})

	t.Run("preserves explicitly empty values", func(t *testing.T) {
		t.Setenv(key, "")
		if value := getEnv(key, "fallback"); value != "" {
			t.Fatalf("expected explicit empty value, got %q", value)
		}
	})

	t.Run("returns non-empty value", func(t *testing.T) {
		t.Setenv(key, "value")
		if value := getEnv(key, "fallback"); value != "value" {
			t.Fatalf("expected env value, got %q", value)
		}
	})
}
