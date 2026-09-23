package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveConfigPathWithFallback(t *testing.T) {
	t.Run("canonical path wins when present", func(t *testing.T) {
		dir := t.TempDir()
		canonical := filepath.Join(dir, "txnode", "config.yml")
		legacy := filepath.Join(dir, "xboard-node", "config.yml")
		mustWriteConfigPath(t, canonical)
		mustWriteConfigPath(t, legacy)

		if got := resolveConfigPathWithFallback(canonical, canonical, legacy); got != canonical {
			t.Fatalf("resolveConfigPathWithFallback() = %q, want canonical %q", got, canonical)
		}
	})

	t.Run("legacy path is fallback when canonical is absent", func(t *testing.T) {
		dir := t.TempDir()
		canonical := filepath.Join(dir, "txnode", "config.yml")
		legacy := filepath.Join(dir, "xboard-node", "config.yml")
		mustWriteConfigPath(t, legacy)

		if got := resolveConfigPathWithFallback(canonical, canonical, legacy); got != legacy {
			t.Fatalf("resolveConfigPathWithFallback() = %q, want legacy %q", got, legacy)
		}
	})

	t.Run("canonical remains selected when neither path exists", func(t *testing.T) {
		dir := t.TempDir()
		canonical := filepath.Join(dir, "txnode", "config.yml")
		legacy := filepath.Join(dir, "xboard-node", "config.yml")

		if got := resolveConfigPathWithFallback(canonical, canonical, legacy); got != canonical {
			t.Fatalf("resolveConfigPathWithFallback() = %q, want canonical %q", got, canonical)
		}
	})

	t.Run("custom config paths never fall back", func(t *testing.T) {
		dir := t.TempDir()
		canonical := filepath.Join(dir, "txnode", "config.yml")
		legacy := filepath.Join(dir, "xboard-node", "config.yml")
		custom := filepath.Join(dir, "custom.yml")
		mustWriteConfigPath(t, legacy)

		if got := resolveConfigPathWithFallback(custom, canonical, legacy); got != custom {
			t.Fatalf("resolveConfigPathWithFallback() = %q, want custom %q", got, custom)
		}
	})
}

func mustWriteConfigPath(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("test: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}
