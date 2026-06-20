package ard

import (
	"encoding/json"
	"strings"
	"testing"
)

func validCatalog() Catalog {
	return Catalog{
		SpecVersion: "1.0",
		Entries: []CatalogEntry{
			{
				Identifier:  "urn:air:acme.com:server:weather",
				DisplayName: "Weather Data Node",
				Type:        TypeMCPServerCard,
				URL:         "https://api.acme.com/mcp/weather.json",
			},
		},
	}
}

func TestValidateCatalogHostInfo(t *testing.T) {
	catalog := validCatalog()
	catalog.Host = &HostInfo{
		DisplayName:      "Acme Registry",
		Identifier:       "did:web:acme.com",
		DocumentationURL: "https://acme.com/docs/ai-catalog",
		LogoURL:          "https://acme.com/logo.png",
		TrustManifest: map[string]any{
			"identity":     "https://registry.acme.com/security",
			"identityType": "https",
		},
	}
	if err := ValidateCatalog(catalog); err != nil {
		t.Fatalf("expected valid host: %v", err)
	}

	catalog.Host.DisplayName = " "
	if err := ValidateCatalog(catalog); err == nil || !strings.Contains(err.Error(), "host: displayName") {
		t.Fatalf("expected missing host displayName to be rejected, got %v", err)
	}

	catalog.Host.DisplayName = "Acme Registry"
	catalog.Host.DocumentationURL = "/docs"
	if err := ValidateCatalog(catalog); err == nil || !strings.Contains(err.Error(), "host: documentationUrl") {
		t.Fatalf("expected relative host documentationUrl to be rejected, got %v", err)
	}

	catalog.Host.DocumentationURL = "https://acme.com/docs/ai-catalog"
	catalog.Host.TrustManifest["identityType"] = "x509"
	if err := ValidateCatalog(catalog); err == nil || !strings.Contains(err.Error(), "trustManifest.identityType") {
		t.Fatalf("expected invalid host trustManifest to be rejected, got %v", err)
	}
}

func TestValidateCatalogEntryRequiresAirURN(t *testing.T) {
	entry := CatalogEntry{
		Identifier:  "urn:air:acme.com:server:weather",
		DisplayName: "Weather Data Node",
		Type:        TypeMCPServerCard,
		URL:         "https://api.acme.com/mcp/weather.json",
	}

	if err := ValidateCatalogEntry(entry); err != nil {
		t.Fatalf("expected valid entry: %v", err)
	}
}

func TestValidateCatalogEntryRejectsLegacyAIURN(t *testing.T) {
	entry := CatalogEntry{
		Identifier:  "urn:ai:acme.com:server:weather",
		DisplayName: "Weather Data Node",
		Type:        TypeMCPServerCard,
		URL:         "https://api.acme.com/mcp/weather.json",
	}

	if err := ValidateCatalogEntry(entry); err == nil {
		t.Fatal("expected legacy urn:ai identifier to be rejected")
	}
}

func TestValidateIdentifier(t *testing.T) {
	if err := ValidateIdentifier("urn:air:acme.com:server:weather"); err != nil {
		t.Fatalf("expected identifier to validate: %v", err)
	}
	if err := ValidateIdentifier("weather"); err == nil {
		t.Fatal("expected invalid identifier to be rejected")
	}
}

func TestValidateCatalogEntryEnforcesValueOrReference(t *testing.T) {
	entry := CatalogEntry{
		Identifier:  "urn:air:acme.com:server:weather",
		DisplayName: "Weather Data Node",
		Type:        TypeMCPServerCard,
		URL:         "https://api.acme.com/mcp/weather.json",
		Data:        map[string]any{"name": "weather"},
	}

	if err := ValidateCatalogEntry(entry); err == nil {
		t.Fatal("expected entry with both url and data to be rejected")
	}
}

func TestValidateCatalogEntryURLRequiresHTTPURL(t *testing.T) {
	entry := CatalogEntry{
		Identifier:  "urn:air:acme.com:server:weather",
		DisplayName: "Weather Data Node",
		Type:        TypeMCPServerCard,
		URL:         "http://127.0.0.1:8080/mcp/weather.json",
	}
	if err := ValidateCatalogEntry(entry); err != nil {
		t.Fatalf("expected http URL to validate: %v", err)
	}

	entry.URL = "/mcp/weather.json"
	if err := ValidateCatalogEntry(entry); err == nil {
		t.Fatal("expected relative URL to be rejected")
	}

	entry.URL = "urn:air:acme.com:server:weather"
	if err := ValidateCatalogEntry(entry); err == nil {
		t.Fatal("expected non-HTTP URL to be rejected")
	}

	entry.URL = "https:///mcp/weather.json"
	if err := ValidateCatalogEntry(entry); err == nil {
		t.Fatal("expected URL without host to be rejected")
	}
}

func TestValidateCatalogEntryUpdatedAtAndMetadata(t *testing.T) {
	entry := CatalogEntry{
		Identifier:  "urn:air:acme.com:server:weather",
		DisplayName: "Weather Data Node",
		Type:        TypeMCPServerCard,
		URL:         "https://api.acme.com/mcp/weather.json",
		UpdatedAt:   "2026-06-21T02:30:00Z",
		Metadata: map[string]any{
			"adapter":     "mcp",
			"remoteCount": 2,
			"verified":    true,
			"deprecated":  nil,
		},
	}
	if err := ValidateCatalogEntry(entry); err != nil {
		t.Fatalf("expected valid updatedAt and metadata: %v", err)
	}

	entry.UpdatedAt = "2026-06-21"
	if err := ValidateCatalogEntry(entry); err == nil {
		t.Fatal("expected date-only updatedAt to be rejected")
	}

	entry.UpdatedAt = "2026-06-21T02:30:00Z"
	entry.Metadata["nested"] = map[string]any{"key": "value"}
	if err := ValidateCatalogEntry(entry); err == nil {
		t.Fatal("expected object metadata value to be rejected")
	}

	delete(entry.Metadata, "nested")
	entry.Metadata["list"] = []any{"value"}
	if err := ValidateCatalogEntry(entry); err == nil {
		t.Fatal("expected array metadata value to be rejected")
	}
}

func TestValidateCatalogEntryTrustManifest(t *testing.T) {
	entry := CatalogEntry{
		Identifier:  "urn:air:acme.com:server:weather",
		DisplayName: "Weather Data Node",
		Type:        TypeMCPServerCard,
		URL:         "https://api.acme.com/mcp/weather.json",
		TrustManifest: map[string]any{
			"sourceDigest": "sha256:abc",
		},
	}
	if err := ValidateCatalogEntry(entry); err == nil {
		t.Fatal("expected trustManifest without identity to be rejected")
	}
	entry.TrustManifest["identity"] = "https://acme.com"
	if err := ValidateCatalogEntry(entry); err == nil {
		t.Fatal("expected invalid sourceDigest to be rejected")
	}
	entry.TrustManifest["sourceDigest"] = "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if err := ValidateCatalogEntry(entry); err != nil {
		t.Fatalf("expected valid trustManifest: %v", err)
	}
}

func TestValidateCatalogEntryTrustManifestIdentityHost(t *testing.T) {
	validDigest := "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	entry := CatalogEntry{
		Identifier:  "urn:air:acme.com:server:weather",
		DisplayName: "Weather Data Node",
		Type:        TypeMCPServerCard,
		URL:         "https://api.acme.com/mcp/weather.json",
		TrustManifest: map[string]any{
			"identity":     "https://acme.com/security",
			"sourceDigest": validDigest,
		},
	}
	if err := ValidateCatalogEntry(entry); err != nil {
		t.Fatalf("expected matching identity host: %v", err)
	}

	entry.TrustManifest["identity"] = "https://evil.example.com"
	if err := ValidateCatalogEntry(entry); err == nil {
		t.Fatal("expected mismatched identity host to be rejected")
	}

	entry.TrustManifest["identity"] = "did:web:acme.com"
	if err := ValidateCatalogEntry(entry); err != nil {
		t.Fatalf("expected non-URL identity to remain accepted for future resolvers: %v", err)
	}
}

func TestValidateCatalogEntryTrustManifestIdentityType(t *testing.T) {
	entry := CatalogEntry{
		Identifier:  "urn:air:acme.com:server:weather",
		DisplayName: "Weather Data Node",
		Type:        TypeMCPServerCard,
		URL:         "https://api.acme.com/mcp/weather.json",
		TrustManifest: map[string]any{
			"identity":     "https://acme.com/security",
			"identityType": "https",
		},
	}
	if err := ValidateCatalogEntry(entry); err != nil {
		t.Fatalf("expected supported identityType: %v", err)
	}

	entry.TrustManifest["identityType"] = "x509"
	if err := ValidateCatalogEntry(entry); err == nil {
		t.Fatal("expected unsupported identityType to be rejected")
	}

	entry.TrustManifest["identityType"] = 42
	if err := ValidateCatalogEntry(entry); err == nil {
		t.Fatal("expected non-string identityType to be rejected")
	}
}

func TestValidateCatalogEntryTrustManifestSourceDigestType(t *testing.T) {
	entry := CatalogEntry{
		Identifier:  "urn:air:acme.com:server:weather",
		DisplayName: "Weather Data Node",
		Type:        TypeMCPServerCard,
		URL:         "https://api.acme.com/mcp/weather.json",
		TrustManifest: map[string]any{
			"identity":     "https://acme.com/security",
			"sourceDigest": 42,
		},
	}
	if err := ValidateCatalogEntry(entry); err == nil {
		t.Fatal("expected non-string sourceDigest to be rejected")
	}
}

func TestValidateCatalogEntryTrustManifestTrustSchema(t *testing.T) {
	entry := CatalogEntry{
		Identifier:  "urn:air:acme.com:server:weather",
		DisplayName: "Weather Data Node",
		Type:        TypeMCPServerCard,
		URL:         "https://api.acme.com/mcp/weather.json",
		TrustManifest: map[string]any{
			"identity": "https://acme.com/security",
			"trustSchema": map[string]any{
				"identifier":          "urn:air:acme.com:trust:soc2",
				"version":             "1.0",
				"governanceUri":       "https://acme.com/security/governance",
				"verificationMethods": []any{"did", "x509"},
			},
			"signature": "detached-jws-placeholder",
		},
	}
	if err := ValidateCatalogEntry(entry); err != nil {
		t.Fatalf("expected valid trustSchema: %v", err)
	}

	schema := entry.TrustManifest["trustSchema"].(map[string]any)
	delete(schema, "version")
	if err := ValidateCatalogEntry(entry); err == nil {
		t.Fatal("expected trustSchema without version to be rejected")
	}

	schema["version"] = "1.0"
	schema["governanceUri"] = "/security/governance"
	if err := ValidateCatalogEntry(entry); err == nil {
		t.Fatal("expected relative governanceUri to be rejected")
	}

	schema["governanceUri"] = "https://acme.com/security/governance"
	schema["verificationMethods"] = []any{"did", 42}
	if err := ValidateCatalogEntry(entry); err == nil {
		t.Fatal("expected non-string verification method to be rejected")
	}

	schema["verificationMethods"] = []any{"did", "x509"}
	entry.TrustManifest["signature"] = 42
	if err := ValidateCatalogEntry(entry); err == nil {
		t.Fatal("expected non-string signature to be rejected")
	}
}

func TestValidateCatalogEntryTrustManifestAttestations(t *testing.T) {
	validDigest := "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	entry := CatalogEntry{
		Identifier:  "urn:air:acme.com:server:weather",
		DisplayName: "Weather Data Node",
		Type:        TypeMCPServerCard,
		URL:         "https://api.acme.com/mcp/weather.json",
		TrustManifest: map[string]any{
			"identity": "https://acme.com/security",
			"attestations": []any{
				map[string]any{
					"type":      "SOC2-Type2",
					"uri":       "https://acme.com/audits/soc2.pdf",
					"mediaType": "application/pdf",
					"digest":    validDigest,
				},
			},
		},
	}
	if err := ValidateCatalogEntry(entry); err != nil {
		t.Fatalf("expected valid attestations: %v", err)
	}

	entry.TrustManifest["attestations"] = []any{
		map[string]any{
			"type":      "SOC2-Type2",
			"uri":       "/audits/soc2.pdf",
			"mediaType": "application/pdf",
		},
	}
	if err := ValidateCatalogEntry(entry); err == nil {
		t.Fatal("expected relative attestation URI to be rejected")
	}

	entry.TrustManifest["attestations"] = []any{
		map[string]any{
			"type": "SOC2-Type2",
			"uri":  "https://acme.com/audits/soc2.pdf",
		},
	}
	if err := ValidateCatalogEntry(entry); err == nil {
		t.Fatal("expected attestation without mediaType to be rejected")
	}

	entry.TrustManifest["attestations"] = []any{
		map[string]any{
			"type":      "SOC2-Type2",
			"uri":       "https://acme.com/audits/soc2.pdf",
			"mediaType": "application/pdf",
			"digest":    "sha256:abc",
		},
	}
	if err := ValidateCatalogEntry(entry); err == nil {
		t.Fatal("expected invalid attestation digest to be rejected")
	}
}

func TestValidateCatalogEntryTrustManifestProvenance(t *testing.T) {
	validDigest := "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	entry := CatalogEntry{
		Identifier:  "urn:air:acme.com:server:weather",
		DisplayName: "Weather Data Node",
		Type:        TypeMCPServerCard,
		URL:         "https://api.acme.com/mcp/weather.json",
		TrustManifest: map[string]any{
			"identity": "https://acme.com/security",
			"provenance": []any{
				map[string]any{
					"relation":     "publishedFrom",
					"sourceId":     "urn:air:acme.com:source:weather-card",
					"sourceDigest": validDigest,
				},
			},
		},
	}
	if err := ValidateCatalogEntry(entry); err != nil {
		t.Fatalf("expected valid provenance: %v", err)
	}

	entry.TrustManifest["provenance"] = []any{
		map[string]any{
			"relation": "mirroredFrom",
			"sourceId": "urn:air:acme.com:source:weather-card",
		},
	}
	if err := ValidateCatalogEntry(entry); err == nil {
		t.Fatal("expected unsupported provenance relation to be rejected")
	}

	entry.TrustManifest["provenance"] = []any{
		map[string]any{
			"relation": "publishedFrom",
		},
	}
	if err := ValidateCatalogEntry(entry); err == nil {
		t.Fatal("expected provenance without sourceId to be rejected")
	}

	entry.TrustManifest["provenance"] = []any{
		map[string]any{
			"relation":     "publishedFrom",
			"sourceId":     "urn:air:acme.com:source:weather-card",
			"sourceDigest": "sha256:abc",
		},
	}
	if err := ValidateCatalogEntry(entry); err == nil {
		t.Fatal("expected invalid provenance sourceDigest to be rejected")
	}
}

func TestSearchFilterAcceptsScalarAndArray(t *testing.T) {
	var request SearchRequest
	body := []byte(`{
		"query": {
			"text": "weather",
			"filter": {
				"type": "application/mcp-server-card+json",
				"tags": ["tools", "weather"]
			}
		}
	}`)

	if err := json.Unmarshal(body, &request); err != nil {
		t.Fatalf("unmarshal search request: %v", err)
	}
	if got := request.Query.Filter["type"]; len(got) != 1 || got[0] != TypeMCPServerCard {
		t.Fatalf("unexpected type filter: %#v", got)
	}
	if got := request.Query.Filter["tags"]; len(got) != 2 {
		t.Fatalf("unexpected tags filter: %#v", got)
	}
}
