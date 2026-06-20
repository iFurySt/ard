package catalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ifuryst/ard/internal/ard"
)

func TestLoadLocalCatalog(t *testing.T) {
	catalog, err := Load(context.Background(), filepath.Join("testdata", "acme-ai-catalog.json"))
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}
	if catalog.SpecVersion != "1.0" {
		t.Fatalf("unexpected spec version: %s", catalog.SpecVersion)
	}
	if len(catalog.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(catalog.Entries))
	}
	if catalog.Entries[1].Type != ard.TypeMCPServerCard {
		t.Fatalf("expected MCP server card type, got %s", catalog.Entries[1].Type)
	}
}

func TestLoadHTTPCatalog(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "acme-ai-catalog.json"))
	if err != nil {
		t.Fatalf("read test catalog: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write(data)
	}))
	defer server.Close()

	catalog, err := Load(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("load HTTP catalog: %v", err)
	}
	if len(catalog.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(catalog.Entries))
	}
}
