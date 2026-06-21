package cli

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyCatalogRequiresSourceDigests(t *testing.T) {
	catalogPath := filepath.Join(t.TempDir(), "ai-catalog.json")
	if err := os.WriteFile(catalogPath, []byte(`{
  "specVersion": "1.0",
  "entries": [
    {
      "identifier": "urn:air:example.com:server:weather",
      "displayName": "Weather",
      "type": "application/mcp-server-card+json",
      "url": "https://example.com/mcp.json"
    }
  ]
}`), 0o600); err != nil {
		t.Fatalf("write catalog: %v", err)
	}

	command := NewRootCommand()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs([]string{"verify", "catalog", catalogPath, "--require-source-digests"})
	err := command.Execute()
	if err == nil {
		t.Fatal("expected missing sourceDigest to fail")
	}
	if !strings.Contains(err.Error(), "sourceDigest required for url delivery") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyCatalogVerifiesAttestationDigests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		_, _ = response.Write([]byte("soc2-attestation"))
	}))
	t.Cleanup(server.Close)

	catalogPath := filepath.Join(t.TempDir(), "ai-catalog.json")
	if err := os.WriteFile(catalogPath, []byte(`{
  "specVersion": "1.0",
  "entries": [
    {
      "identifier": "urn:air:example.com:agent:weather",
      "displayName": "Weather",
      "type": "application/a2a-agent-card+json",
      "url": "https://example.com/agent-card.json",
      "trustManifest": {
        "identity": "https://example.com",
        "attestations": [
          {
            "type": "SOC2-Type2",
            "uri": "`+server.URL+`",
            "mediaType": "application/pdf",
            "digest": "`+cliTestSHA256("soc2-attestation")+`"
          }
        ]
      }
    }
  ]
}`), 0o600); err != nil {
		t.Fatalf("write catalog: %v", err)
	}

	command := NewRootCommand()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs([]string{"verify", "catalog", catalogPath, "--require-attestation-digests", "--json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("execute verify: %v", err)
	}
	if !strings.Contains(output.String(), `"attestationDigestsVerified": 1`) {
		t.Fatalf("expected attestation digest verification output, got %s", output.String())
	}
}

func TestVerifyCatalogVerifiesProvenanceDigests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		_, _ = response.Write([]byte("source-artifact"))
	}))
	t.Cleanup(server.Close)

	catalogPath := filepath.Join(t.TempDir(), "ai-catalog.json")
	if err := os.WriteFile(catalogPath, []byte(`{
  "specVersion": "1.0",
  "entries": [
    {
      "identifier": "urn:air:example.com:agent:weather",
      "displayName": "Weather",
      "type": "application/a2a-agent-card+json",
      "url": "https://example.com/agent-card.json",
      "trustManifest": {
        "identity": "https://example.com",
        "provenance": [
          {
            "relation": "publishedFrom",
            "sourceId": "`+server.URL+`",
            "sourceDigest": "`+cliTestSHA256("source-artifact")+`"
          }
        ]
      }
    }
  ]
}`), 0o600); err != nil {
		t.Fatalf("write catalog: %v", err)
	}

	command := NewRootCommand()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs([]string{"verify", "catalog", catalogPath, "--require-provenance-digests", "--json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("execute verify: %v", err)
	}
	if !strings.Contains(output.String(), `"provenanceDigestsVerified": 1`) {
		t.Fatalf("expected provenance digest verification output, got %s", output.String())
	}
}

func TestVerifyCatalogEvaluatesPolicy(t *testing.T) {
	tempDir := t.TempDir()
	policyPath := filepath.Join(tempDir, "policy.json")
	if err := os.WriteFile(policyPath, []byte(`{
  "version": "1",
  "requireSourceDigestForURLArtifacts": true
}`), 0o600); err != nil {
		t.Fatalf("write policy: %v", err)
	}
	catalogPath := filepath.Join(tempDir, "ai-catalog.json")
	if err := os.WriteFile(catalogPath, []byte(`{
  "specVersion": "1.0",
  "entries": [
    {
      "identifier": "urn:air:example.com:server:weather",
      "displayName": "Weather",
      "type": "application/mcp-server-card+json",
      "url": "https://example.com/mcp.json",
      "trustManifest": {
        "identity": "https://example.com",
        "sourceDigest": "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
      }
    }
  ]
}`), 0o600); err != nil {
		t.Fatalf("write catalog: %v", err)
	}

	command := NewRootCommand()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs([]string{"--policy-file", policyPath, "verify", "catalog", catalogPath, "--json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("execute verify: %v", err)
	}
	if !strings.Contains(output.String(), `"policyEvaluated": true`) || !strings.Contains(output.String(), `"reason": "default active"`) {
		t.Fatalf("expected policy evaluation output, got %s", output.String())
	}
}

func TestVerifyCatalogRejectsPolicyDeniedCatalog(t *testing.T) {
	tempDir := t.TempDir()
	policyPath := filepath.Join(tempDir, "policy.json")
	if err := os.WriteFile(policyPath, []byte(`{
  "version": "1",
  "requireSourceDigestForURLArtifacts": true
}`), 0o600); err != nil {
		t.Fatalf("write policy: %v", err)
	}
	catalogPath := filepath.Join(tempDir, "ai-catalog.json")
	if err := os.WriteFile(catalogPath, []byte(`{
  "specVersion": "1.0",
  "entries": [
    {
      "identifier": "urn:air:example.com:server:weather",
      "displayName": "Weather",
      "type": "application/mcp-server-card+json",
      "url": "https://example.com/mcp.json",
      "trustManifest": {
        "identity": "https://example.com"
      }
    }
  ]
}`), 0o600); err != nil {
		t.Fatalf("write catalog: %v", err)
	}

	command := NewRootCommand()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs([]string{"--policy-file", policyPath, "verify", "catalog", catalogPath})
	err := command.Execute()
	if err == nil || !strings.Contains(err.Error(), "sourceDigest required for url delivery") {
		t.Fatalf("expected policy denial, got %v output %s", err, output.String())
	}
}

func TestVerifyCatalogVerifiesJWSSignatures(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tempDir := t.TempDir()
	anchorsPath := filepath.Join(tempDir, "anchors.json")
	if err := os.WriteFile(anchorsPath, []byte(`{
  "keys": [
    {
      "kty": "OKP",
      "crv": "Ed25519",
      "kid": "acme-ed25519",
      "alg": "EdDSA",
      "x": "`+base64.RawURLEncoding.EncodeToString(publicKey)+`"
    }
  ]
}`), 0o600); err != nil {
		t.Fatalf("write anchors: %v", err)
	}

	trustManifest := map[string]any{
		"identity":     "https://example.com",
		"identityType": "https",
	}
	trustManifest["signature"] = cliTestDetachedJWS(t, "acme-ed25519", trustManifest, privateKey)
	catalog := map[string]any{
		"specVersion": "1.0",
		"entries": []map[string]any{
			{
				"identifier":    "urn:air:example.com:agent:weather",
				"displayName":   "Weather",
				"type":          "application/a2a-agent-card+json",
				"url":           "https://example.com/agent-card.json",
				"trustManifest": trustManifest,
			},
		},
	}
	catalogData, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		t.Fatalf("marshal catalog: %v", err)
	}
	catalogPath := filepath.Join(tempDir, "ai-catalog.json")
	if err := os.WriteFile(catalogPath, catalogData, 0o600); err != nil {
		t.Fatalf("write catalog: %v", err)
	}

	command := NewRootCommand()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs([]string{"verify", "catalog", catalogPath, "--jws-trust-anchors", anchorsPath, "--require-jws-signatures", "--json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("execute verify: %v", err)
	}
	if !strings.Contains(output.String(), `"signaturesVerified": 1`) {
		t.Fatalf("expected signature verification output, got %s", output.String())
	}
}

func TestVerifyCatalogRejectsNonHTTPSRemoteJWKS(t *testing.T) {
	tempDir := t.TempDir()
	catalogPath := filepath.Join(tempDir, "ai-catalog.json")
	if err := os.WriteFile(catalogPath, []byte(`{
  "specVersion": "1.0",
  "entries": [
    {
      "identifier": "urn:air:example.com:agent:weather",
      "displayName": "Weather",
      "type": "application/a2a-agent-card+json",
      "url": "https://example.com/agent-card.json",
      "trustManifest": {
        "identity": "https://example.com",
        "signature": "placeholder"
      }
    }
  ]
}`), 0o600); err != nil {
		t.Fatalf("write catalog: %v", err)
	}

	command := NewRootCommand()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs([]string{"verify", "catalog", catalogPath, "--jws-remote-jwks", "http://example.com/jwks.json"})
	err := command.Execute()
	if err == nil || !strings.Contains(err.Error(), "remote JWKS URL must be absolute HTTPS") {
		t.Fatalf("expected remote JWKS HTTPS error, got %v output %s", err, output.String())
	}
}

func TestVerifyCatalogDiscoverDIDWebRequiresDIDWebIdentity(t *testing.T) {
	tempDir := t.TempDir()
	catalogPath := filepath.Join(tempDir, "ai-catalog.json")
	if err := os.WriteFile(catalogPath, []byte(`{
  "specVersion": "1.0",
  "entries": [
    {
      "identifier": "urn:air:example.com:agent:weather",
      "displayName": "Weather",
      "type": "application/a2a-agent-card+json",
      "url": "https://example.com/agent-card.json",
      "trustManifest": {
        "identity": "did:key:z6Mkh",
        "identityType": "did",
        "signature": "placeholder"
      }
    }
  ]
}`), 0o600); err != nil {
		t.Fatalf("write catalog: %v", err)
	}

	command := NewRootCommand()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs([]string{"verify", "catalog", catalogPath, "--jws-discover-did-web"})
	err := command.Execute()
	if err == nil || !strings.Contains(err.Error(), "--jws-discover-did-web") {
		t.Fatalf("expected did:web discovery anchor error, got %v output %s", err, output.String())
	}
}

func TestVerifyCatalogDiscoverOIDCRequiresHTTPSIdentity(t *testing.T) {
	tempDir := t.TempDir()
	catalogPath := filepath.Join(tempDir, "ai-catalog.json")
	if err := os.WriteFile(catalogPath, []byte(`{
  "specVersion": "1.0",
  "entries": [
    {
      "identifier": "urn:air:example.com:agent:weather",
      "displayName": "Weather",
      "type": "application/a2a-agent-card+json",
      "url": "https://example.com/agent-card.json",
      "trustManifest": {
        "identity": "did:key:z6Mkh",
        "identityType": "did",
        "signature": "placeholder"
      }
    }
  ]
}`), 0o600); err != nil {
		t.Fatalf("write catalog: %v", err)
	}

	command := NewRootCommand()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs([]string{"verify", "catalog", catalogPath, "--jws-discover-oidc"})
	err := command.Execute()
	if err == nil || !strings.Contains(err.Error(), "--jws-discover-oidc") {
		t.Fatalf("expected OIDC discovery anchor error, got %v output %s", err, output.String())
	}
}

func TestVerifyCatalogDiscoverTLSCertRequiresHTTPSIdentity(t *testing.T) {
	tempDir := t.TempDir()
	catalogPath := filepath.Join(tempDir, "ai-catalog.json")
	if err := os.WriteFile(catalogPath, []byte(`{
  "specVersion": "1.0",
  "entries": [
    {
      "identifier": "urn:air:example.com:agent:weather",
      "displayName": "Weather",
      "type": "application/a2a-agent-card+json",
      "url": "https://example.com/agent-card.json",
      "trustManifest": {
        "identity": "did:key:z6Mkh",
        "identityType": "did",
        "signature": "placeholder"
      }
    }
  ]
}`), 0o600); err != nil {
		t.Fatalf("write catalog: %v", err)
	}

	command := NewRootCommand()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs([]string{"verify", "catalog", catalogPath, "--jws-discover-tls-cert"})
	err := command.Execute()
	if err == nil || !strings.Contains(err.Error(), "--jws-discover-tls-cert") {
		t.Fatalf("expected TLS certificate discovery anchor error, got %v output %s", err, output.String())
	}
}

func cliTestSHA256(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func cliTestDetachedJWS(t *testing.T, keyID string, trustManifest map[string]any, privateKey ed25519.PrivateKey) string {
	t.Helper()
	protected, err := json.Marshal(map[string]string{
		"alg": "EdDSA",
		"kid": keyID,
	})
	if err != nil {
		t.Fatalf("marshal protected header: %v", err)
	}
	protectedPart := base64.RawURLEncoding.EncodeToString(protected)
	payload := make(map[string]any, len(trustManifest))
	for key, value := range trustManifest {
		if key != "signature" {
			payload[key] = value
		}
	}
	payloadData, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	signingInput := []byte(protectedPart + "." + base64.RawURLEncoding.EncodeToString(payloadData))
	signature := ed25519.Sign(privateKey, signingInput)
	return protectedPart + ".." + base64.RawURLEncoding.EncodeToString(signature)
}
