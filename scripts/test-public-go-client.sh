#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
workdir="$(mktemp -d /tmp/ard-public-go-client-XXXXXX)"
cleanup() {
  rm -rf "${workdir}"
}
trap cleanup EXIT

cd "${workdir}"
go mod init example.com/ard-public-go-client-check >/dev/null
go mod edit -require github.com/ifuryst/ard@v0.0.0
go mod edit -replace "github.com/ifuryst/ard=${repo_root}"

cat >client_test.go <<'GO'
package ardpublicclientcheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ifuryst/ard/pkg/ard"
	"github.com/ifuryst/ard/pkg/client"
)

func TestPublicGoClientImportsFromExternalModule(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/search":
			_, _ = response.Write([]byte(`{"results":[{"identifier":"urn:air:example.com:server:weather","displayName":"Weather","type":"application/mcp-server-card+json","url":"https://example.com/mcp.json","score":100,"source":"local"}]}`))
		case "/agents":
			_, _ = response.Write([]byte(`{"items":[{"identifier":"urn:air:example.com:server:weather","displayName":"Weather","type":"application/mcp-server-card+json","url":"https://example.com/mcp.json"}],"total":1}`))
		default:
			http.NotFound(response, request)
		}
	}))
	defer server.Close()

	registry, err := client.New(server.URL)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	results, err := registry.Search(context.Background(), ard.SearchRequest{
		Query: ard.SearchQuery{Text: "weather"},
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results.Results) != 1 || results.Results[0].Type != ard.TypeMCPServerCard {
		t.Fatalf("unexpected search response: %#v", results)
	}
	list, err := registry.Browse(context.Background(), client.BrowseOptions{PageSize: 1})
	if err != nil {
		t.Fatalf("browse: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("unexpected browse response: %#v", list)
	}
}
GO

go test ./...
