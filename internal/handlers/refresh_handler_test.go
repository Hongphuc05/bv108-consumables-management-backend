package handlers

import (
	"path/filepath"
	"testing"
)

func TestResolveExistingDirectoryReturnsFirstExistingUniquePath(t *testing.T) {
	t.Parallel()

	firstDir := t.TempDir()
	secondDir := t.TempDir()

	resolved, tried, err := resolveExistingDirectory([]ubotDirCandidate{
		{label: "missing", path: filepath.Join(firstDir, "does-not-exist")},
		{label: "first", path: firstDir},
		{label: "duplicate-first", path: firstDir},
		{label: "second", path: secondDir},
	})
	if err != nil {
		t.Fatalf("resolveExistingDirectory returned unexpected error: %v", err)
	}

	if resolved != filepath.Clean(firstDir) {
		t.Fatalf("resolveExistingDirectory resolved %q, want %q", resolved, filepath.Clean(firstDir))
	}

	if len(tried) != 2 {
		t.Fatalf("resolveExistingDirectory tried %d paths, want 2 unique paths", len(tried))
	}
}

func TestResolveExistingDirectoryReturnsErrorWhenNoCandidateExists(t *testing.T) {
	t.Parallel()

	missingDir := filepath.Join(t.TempDir(), "missing")
	_, tried, err := resolveExistingDirectory([]ubotDirCandidate{
		{label: "missing", path: missingDir},
	})
	if err == nil {
		t.Fatal("resolveExistingDirectory returned nil error, want error")
	}

	if len(tried) != 1 {
		t.Fatalf("resolveExistingDirectory tried %d paths, want 1", len(tried))
	}
}

func TestResolveUBotDirPrefersEnvOverride(t *testing.T) {
	customDir := t.TempDir()
	t.Setenv(ubotAPIDirEnvKey, customDir)

	resolved, _, err := resolveUBotDir()
	if err != nil {
		t.Fatalf("resolveUBotDir returned unexpected error: %v", err)
	}

	if resolved != filepath.Clean(customDir) {
		t.Fatalf("resolveUBotDir resolved %q, want %q", resolved, filepath.Clean(customDir))
	}
}
