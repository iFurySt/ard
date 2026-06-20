package adapters

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ifuryst/ard/internal/ard"
)

func TestLoadMCPServerCardFromLocalFile(t *testing.T) {
	entry, err := LoadMCPServerCard(
		context.Background(),
		filepath.Join("testdata", "mcp-server-card.json"),
		Options{},
	)
	if err != nil {
		t.Fatalf("load MCP server card: %v", err)
	}
	if entry.Type != ard.TypeMCPServerCard {
		t.Fatalf("unexpected type: %s", entry.Type)
	}
	if entry.Identifier != "urn:air:agent.localhost:server:agentmemory-mcp" {
		t.Fatalf("unexpected identifier: %s", entry.Identifier)
	}
	if entry.URL != "" || entry.Data == nil {
		t.Fatalf("local artifact should be embedded as data")
	}
	if !containsString(entry.Capabilities, "remote:streamable-http") {
		t.Fatalf("expected remote transport capability, got %#v", entry.Capabilities)
	}
}

func TestLoadA2AAgentCardFromHTTP(t *testing.T) {
	source := testArtifactServer(t, filepath.Join("testdata", "a2a-agent-card.json"))
	entry, err := LoadA2AAgentCard(
		context.Background(),
		source,
		Options{Publisher: "example.com"},
	)
	if err != nil {
		t.Fatalf("load A2A agent card: %v", err)
	}
	if entry.Identifier != "urn:air:example.com:agent:hello-world-agent" {
		t.Fatalf("unexpected identifier: %s", entry.Identifier)
	}
	if entry.URL != source || entry.Data != nil {
		t.Fatalf("HTTP artifact should be referenced by URL")
	}
	if !containsString(entry.Capabilities, "hello_world") {
		t.Fatalf("expected skill capability, got %#v", entry.Capabilities)
	}
	if got := entry.Metadata["protocolVersion"]; got != "0.3.0" {
		t.Fatalf("unexpected protocolVersion metadata: %#v", got)
	}
}

func TestLoadSkillFromLocalFileWithIdentifierOverride(t *testing.T) {
	entry, err := LoadSkill(
		context.Background(),
		filepath.Join("testdata", "open-browser-use", "SKILL.md"),
		Options{Identifier: "urn:air:github.com:ifuryst:open-browser-use"},
	)
	if err != nil {
		t.Fatalf("load skill: %v", err)
	}
	if entry.Type != ard.TypeAISkill {
		t.Fatalf("unexpected type: %s", entry.Type)
	}
	if entry.DisplayName != "open-browser-use" {
		t.Fatalf("unexpected displayName: %s", entry.DisplayName)
	}
	if entry.URL != "" || entry.Data == nil {
		t.Fatalf("local skill should be embedded as data")
	}
	if got := entry.Data["description"]; got == "" {
		t.Fatalf("expected skill description in embedded data")
	}
}

func testArtifactServer(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read test artifact: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write(data)
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
