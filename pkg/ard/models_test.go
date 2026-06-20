package ard

import "testing"

func TestPublicModelAliasesValidateCatalogEntry(t *testing.T) {
	entry := CatalogEntry{
		Identifier:  "urn:air:example.com:server:weather",
		DisplayName: "Weather",
		Type:        TypeMCPServerCard,
		URL:         "https://example.com/mcp.json",
	}
	if err := ValidateCatalogEntry(entry); err != nil {
		t.Fatalf("validate catalog entry: %v", err)
	}
	if publisher := Publisher(entry.Identifier); publisher != "example.com" {
		t.Fatalf("unexpected publisher: %s", publisher)
	}
}
