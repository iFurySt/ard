package adapters

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ifuryst/ard/internal/ard"
	"github.com/ifuryst/ard/internal/requestid"
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

func TestLoadA2AAgentCardFromHTTPPropagatesRequestID(t *testing.T) {
	seenRequestID := ""
	source := testArtifactServerWithHandler(t, filepath.Join("testdata", "a2a-agent-card.json"), func(request *http.Request) {
		seenRequestID = request.Header.Get(requestid.Header)
	})
	entry, err := LoadA2AAgentCard(
		requestid.With(context.Background(), "artifact-loader-request"),
		source,
		Options{Publisher: "example.com"},
	)
	if err != nil {
		t.Fatalf("load A2A agent card: %v", err)
	}
	if entry.Identifier != "urn:air:example.com:agent:hello-world-agent" {
		t.Fatalf("unexpected identifier: %s", entry.Identifier)
	}
	if seenRequestID != "artifact-loader-request" {
		t.Fatalf("expected request ID propagation, got %q", seenRequestID)
	}
}

func TestLoadA2AAgentCardPinsSourceDigest(t *testing.T) {
	source := testArtifactServer(t, filepath.Join("testdata", "a2a-agent-card.json"))
	entry, err := LoadA2AAgentCard(
		context.Background(),
		source,
		Options{Publisher: "example.com", PinSourceDigest: true},
	)
	if err != nil {
		t.Fatalf("load A2A agent card: %v", err)
	}
	if entry.TrustManifest["identity"] != "https://example.com" {
		t.Fatalf("unexpected trust identity: %#v", entry.TrustManifest)
	}
	sourceDigest, _ := entry.TrustManifest["sourceDigest"].(string)
	if sourceDigest == "" {
		t.Fatalf("expected sourceDigest: %#v", entry.TrustManifest)
	}
}

func TestPinSourceDigestRequiresURL(t *testing.T) {
	_, err := LoadMCPServerCard(
		context.Background(),
		filepath.Join("testdata", "mcp-server-card.json"),
		Options{PinSourceDigest: true},
	)
	if err == nil {
		t.Fatal("expected local source pinning to fail")
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

func TestLoadOpenAPIFromLocalYAML(t *testing.T) {
	entry, err := LoadOpenAPI(
		context.Background(),
		filepath.Join("testdata", "openapi-weather.yaml"),
		Options{},
	)
	if err != nil {
		t.Fatalf("load OpenAPI document: %v", err)
	}
	if entry.Type != ard.TypeOpenAPI {
		t.Fatalf("unexpected type: %s", entry.Type)
	}
	if entry.Identifier != "urn:air:agent.localhost:api:weather-forecast-api" {
		t.Fatalf("unexpected identifier: %s", entry.Identifier)
	}
	if entry.DisplayName != "Weather Forecast API" {
		t.Fatalf("unexpected displayName: %s", entry.DisplayName)
	}
	if entry.URL != "" || entry.Data == nil {
		t.Fatalf("local OpenAPI document should be embedded as data")
	}
	if !containsString(entry.Capabilities, "getCurrentWeather") {
		t.Fatalf("expected operationId capability, got %#v", entry.Capabilities)
	}
	if got := entry.Metadata["apiFormat"]; got != "3.1.0" {
		t.Fatalf("unexpected apiFormat metadata: %#v", got)
	}
}

func testArtifactServer(t *testing.T, path string) string {
	return testArtifactServerWithHandler(t, path, nil)
}

func testArtifactServerWithHandler(t *testing.T, path string, inspect func(*http.Request)) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read test artifact: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if inspect != nil {
			inspect(request)
		}
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
