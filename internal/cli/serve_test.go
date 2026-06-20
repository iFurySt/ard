package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAdminTokensCombinesLegacyAndFileTokens(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tokens.json")
	if err := os.WriteFile(path, []byte(`{
  "version": "1",
  "tokens": [
    {"name": "reader", "token": "reader-token", "role": "reader"}
  ]
}`), 0o600); err != nil {
		t.Fatalf("write tokens file: %v", err)
	}

	tokens, err := loadAdminTokens(&rootOptions{
		adminToken:      "admin-token",
		adminTokensFile: path,
	})
	if err != nil {
		t.Fatalf("load admin tokens: %v", err)
	}
	if len(tokens) != 2 {
		t.Fatalf("expected 2 tokens, got %d", len(tokens))
	}
	if tokens[0].Name != "default-admin" || tokens[0].Role != "admin" {
		t.Fatalf("unexpected legacy token: %#v", tokens[0])
	}
	if tokens[1].Name != "reader" || tokens[1].Role != "reader" {
		t.Fatalf("unexpected file token: %#v", tokens[1])
	}
}

func TestLoadAdminAuthConfigKeepsFileTokensReloadable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tokens.json")
	if err := os.WriteFile(path, []byte(`{
  "version": "1",
  "tokens": [
    {"name": "reader", "token": "reader-token", "role": "reader"}
  ]
}`), 0o600); err != nil {
		t.Fatalf("write tokens file: %v", err)
	}

	tokens, tokensFile, err := loadAdminAuthConfig(&rootOptions{
		adminToken:      "admin-token",
		adminTokensFile: path,
	})
	if err != nil {
		t.Fatalf("load admin auth config: %v", err)
	}
	if tokensFile != path {
		t.Fatalf("expected tokens file %s, got %s", path, tokensFile)
	}
	if len(tokens) != 1 || tokens[0].Name != "default-admin" || tokens[0].Role != "admin" {
		t.Fatalf("expected only static legacy token, got %#v", tokens)
	}
}
