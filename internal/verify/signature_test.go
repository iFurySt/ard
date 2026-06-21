package verify

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ifuryst/ard/internal/ard"
)

func TestVerifySignatures(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	trustManifest := map[string]any{
		"identity":     "https://example.com",
		"identityType": "https",
	}
	trustManifest["signature"] = testDetachedJWS(t, "acme-ed25519", trustManifest, privateKey)

	results, err := VerifySignatures(signedCatalog(trustManifest), SignatureOptions{
		TrustAnchors: TrustAnchors{Keys: []TrustAnchorKey{
			{
				KeyID:     "acme-ed25519",
				Algorithm: "EdDSA",
				PublicKey: base64.RawURLEncoding.EncodeToString(publicKey),
			},
		}},
	})
	if err != nil {
		t.Fatalf("verify signature: %v", err)
	}
	if len(results) != 1 || !results[0].Verified || results[0].KeyID != "acme-ed25519" {
		t.Fatalf("unexpected results: %#v", results)
	}
}

func TestLoadTrustAnchorsAcceptsJWKSOKPEd25519(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	anchorsPath := filepath.Join(t.TempDir(), "jwks.json")
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
		t.Fatalf("write JWKS: %v", err)
	}

	anchors, err := LoadTrustAnchors(anchorsPath)
	if err != nil {
		t.Fatalf("load JWKS trust anchors: %v", err)
	}
	trustManifest := map[string]any{
		"identity": "https://example.com",
	}
	trustManifest["signature"] = testDetachedJWS(t, "acme-ed25519", trustManifest, privateKey)
	results, err := VerifySignatures(signedCatalog(trustManifest), SignatureOptions{
		TrustAnchors: anchors,
	})
	if err != nil {
		t.Fatalf("verify signature with JWKS trust anchors: %v", err)
	}
	if len(results) != 1 || !results[0].Verified {
		t.Fatalf("unexpected results: %#v", results)
	}
}

func TestLoadTrustAnchorsRejectsUnsupportedJWKSKey(t *testing.T) {
	anchorsPath := filepath.Join(t.TempDir(), "jwks.json")
	if err := os.WriteFile(anchorsPath, []byte(`{
  "keys": [
    {
      "kty": "OKP",
      "crv": "X25519",
      "kid": "acme-x25519",
      "alg": "EdDSA",
      "x": "abc"
    }
  ]
}`), 0o600); err != nil {
		t.Fatalf("write JWKS: %v", err)
	}

	_, err := LoadTrustAnchors(anchorsPath)
	if err == nil || !strings.Contains(err.Error(), "JWKS crv must be Ed25519") {
		t.Fatalf("expected unsupported JWKS curve error, got %v", err)
	}
}

func TestLoadRemoteTrustAnchorsVerifiesMatchingTrustDomain(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/jwk-set+json")
		_, _ = response.Write([]byte(`{
  "keys": [
    {
      "kty": "OKP",
      "crv": "Ed25519",
      "kid": "remote-ed25519",
      "alg": "EdDSA",
      "x": "` + base64.RawURLEncoding.EncodeToString(publicKey) + `"
    }
  ]
}`))
	}))
	t.Cleanup(server.Close)

	anchors, err := LoadRemoteTrustAnchors(context.Background(), server.URL+"/jwks.json", server.Client())
	if err != nil {
		t.Fatalf("load remote JWKS: %v", err)
	}
	host := mustURLHost(t, server.URL)
	trustManifest := map[string]any{
		"identity":     "https://" + host + "/security",
		"identityType": "https",
	}
	trustManifest["signature"] = testDetachedJWS(t, "remote-ed25519", trustManifest, privateKey)
	results, err := VerifySignatures(signedCatalog(trustManifest), SignatureOptions{
		TrustAnchors: anchors,
	})
	if err != nil {
		t.Fatalf("verify signature with remote JWKS: %v", err)
	}
	if len(results) != 1 || !results[0].Verified || results[0].KeySource != server.URL+"/jwks.json" {
		t.Fatalf("unexpected results: %#v", results)
	}
}

func TestLoadRemoteTrustAnchorsRejectsNonHTTPSURL(t *testing.T) {
	_, err := LoadRemoteTrustAnchors(context.Background(), "http://example.com/jwks.json", nil)
	if err == nil || !strings.Contains(err.Error(), "remote JWKS URL must be absolute HTTPS") {
		t.Fatalf("expected HTTPS requirement, got %v", err)
	}
}

func TestDiscoverOIDCTrustAnchorsVerifiesProviderJWKS(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/issuer/.well-known/openid-configuration":
			response.Header().Set("Content-Type", "application/json")
			_, _ = response.Write([]byte(`{
  "issuer": "https://` + request.Host + `/issuer",
  "jwks_uri": "https://` + request.Host + `/jwks.json"
}`))
		case "/jwks.json":
			response.Header().Set("Content-Type", "application/jwk-set+json")
			_, _ = response.Write([]byte(`{
  "keys": [
    {
      "kty": "OKP",
      "crv": "Ed25519",
      "kid": "oidc-ed25519",
      "alg": "EdDSA",
      "x": "` + base64.RawURLEncoding.EncodeToString(publicKey) + `"
    }
  ]
}`))
		default:
			t.Fatalf("unexpected OIDC discovery path: %s", request.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	identity := server.URL + "/issuer"
	trustManifest := map[string]any{
		"identity":     identity,
		"identityType": "https",
	}
	trustManifest["signature"] = testDetachedJWS(t, "oidc-ed25519", trustManifest, privateKey)
	catalog := signedCatalog(trustManifest)
	anchors, err := DiscoverOIDCTrustAnchors(context.Background(), catalog, server.Client())
	if err != nil {
		t.Fatalf("discover OIDC trust anchors: %v", err)
	}
	results, err := VerifySignatures(catalog, SignatureOptions{TrustAnchors: anchors})
	if err != nil {
		t.Fatalf("verify signature with OIDC trust anchors: %v", err)
	}
	if len(results) != 1 || !results[0].Verified || results[0].KeyID != "oidc-ed25519" || results[0].KeySource != server.URL+"/jwks.json" {
		t.Fatalf("unexpected results: %#v", results)
	}
}

func TestDiscoverOIDCTrustAnchorsRejectsIssuerMismatch(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{
  "issuer": "https://evil.example/issuer",
  "jwks_uri": "https://` + request.Host + `/jwks.json"
}`))
	}))
	t.Cleanup(server.Close)

	identity := server.URL + "/issuer"
	trustManifest := map[string]any{
		"identity":     identity,
		"identityType": "https",
	}
	trustManifest["signature"] = testDetachedJWS(t, "oidc-ed25519", trustManifest, privateKey)
	_, err = DiscoverOIDCTrustAnchors(context.Background(), signedCatalog(trustManifest), server.Client())
	if err == nil || !strings.Contains(err.Error(), "OIDC issuer") {
		t.Fatalf("expected issuer mismatch, got %v", err)
	}
}

func TestOIDCConfigurationURL(t *testing.T) {
	tests := []struct {
		identity string
		want     string
	}{
		{
			identity: "https://example.com",
			want:     "https://example.com/.well-known/openid-configuration",
		},
		{
			identity: "https://example.com/tenant",
			want:     "https://example.com/tenant/.well-known/openid-configuration",
		},
		{
			identity: "https://example.com/tenant/",
			want:     "https://example.com/tenant/.well-known/openid-configuration",
		},
	}
	for _, tt := range tests {
		_, got, _, ok, err := oidcConfigurationURL(tt.identity)
		if err != nil {
			t.Fatalf("OIDC configuration URL for %s: %v", tt.identity, err)
		}
		if !ok || got != tt.want {
			t.Fatalf("OIDC configuration URL for %s = %q, %v; want %q, true", tt.identity, got, ok, tt.want)
		}
	}
}

func TestDiscoverDIDWebTrustAnchorsVerifiesDIDDocumentKey(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/.well-known/did.json" {
			t.Fatalf("unexpected DID document path: %s", request.URL.Path)
		}
		host := request.Host
		identity := "did:web:" + strings.ReplaceAll(host, ":", "%3A")
		response.Header().Set("Content-Type", "application/did+json")
		_, _ = response.Write([]byte(`{
  "id": "` + identity + `",
  "verificationMethod": [
    {
      "id": "` + identity + `#key-1",
      "type": "JsonWebKey2020",
      "controller": "` + identity + `",
      "publicKeyJwk": {
        "kty": "OKP",
        "crv": "Ed25519",
        "alg": "EdDSA",
        "x": "` + base64.RawURLEncoding.EncodeToString(publicKey) + `"
      }
    }
  ]
}`))
	}))
	t.Cleanup(server.Close)

	host := mustURLAuthority(t, server.URL)
	identity := "did:web:" + strings.ReplaceAll(host, ":", "%3A")
	keyID := identity + "#key-1"
	trustManifest := map[string]any{
		"identity":     identity,
		"identityType": "did",
	}
	trustManifest["signature"] = testDetachedJWS(t, keyID, trustManifest, privateKey)
	catalog := signedCatalog(trustManifest)
	anchors, err := DiscoverDIDWebTrustAnchors(context.Background(), catalog, server.Client())
	if err != nil {
		t.Fatalf("discover did:web trust anchors: %v", err)
	}
	results, err := VerifySignatures(catalog, SignatureOptions{TrustAnchors: anchors})
	if err != nil {
		t.Fatalf("verify signature with did:web trust anchors: %v", err)
	}
	if len(results) != 1 || !results[0].Verified || results[0].KeyID != keyID {
		t.Fatalf("unexpected results: %#v", results)
	}
}

func TestDiscoverDIDWebTrustAnchorsRejectsControllerMismatch(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		host := request.Host
		identity := "did:web:" + strings.ReplaceAll(host, ":", "%3A")
		response.Header().Set("Content-Type", "application/did+json")
		_, _ = response.Write([]byte(`{
  "id": "` + identity + `",
  "verificationMethod": [
    {
      "id": "` + identity + `#key-1",
      "controller": "did:web:evil.example",
      "publicKeyJwk": {
        "kty": "OKP",
        "crv": "Ed25519",
        "alg": "EdDSA",
        "x": "` + base64.RawURLEncoding.EncodeToString(publicKey) + `"
      }
    }
  ]
}`))
	}))
	t.Cleanup(server.Close)

	host := mustURLAuthority(t, server.URL)
	identity := "did:web:" + strings.ReplaceAll(host, ":", "%3A")
	trustManifest := map[string]any{
		"identity":     identity,
		"identityType": "did",
	}
	trustManifest["signature"] = testDetachedJWS(t, identity+"#key-1", trustManifest, privateKey)
	_, err = DiscoverDIDWebTrustAnchors(context.Background(), signedCatalog(trustManifest), server.Client())
	if err == nil || !strings.Contains(err.Error(), "controller") {
		t.Fatalf("expected controller mismatch, got %v", err)
	}
}

func TestDIDWebDocumentURL(t *testing.T) {
	tests := []struct {
		identity string
		want     string
	}{
		{
			identity: "did:web:example.com",
			want:     "https://example.com/.well-known/did.json",
		},
		{
			identity: "did:web:example.com:users:alice",
			want:     "https://example.com/users/alice/did.json",
		},
		{
			identity: "did:web:localhost%3A8443",
			want:     "https://localhost:8443/.well-known/did.json",
		},
	}
	for _, tt := range tests {
		got, _, ok, err := didWebDocumentURL(tt.identity)
		if err != nil {
			t.Fatalf("did web document URL for %s: %v", tt.identity, err)
		}
		if !ok || got != tt.want {
			t.Fatalf("did web document URL for %s = %q, %v; want %q, true", tt.identity, got, ok, tt.want)
		}
	}
}

func TestVerifySignaturesRejectsRemoteJWKSHostMismatch(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/jwk-set+json")
		_, _ = response.Write([]byte(`{
  "keys": [
    {
      "kty": "OKP",
      "crv": "Ed25519",
      "kid": "remote-ed25519",
      "alg": "EdDSA",
      "x": "` + base64.RawURLEncoding.EncodeToString(publicKey) + `"
    }
  ]
}`))
	}))
	t.Cleanup(server.Close)

	anchors, err := LoadRemoteTrustAnchors(context.Background(), server.URL+"/jwks.json", server.Client())
	if err != nil {
		t.Fatalf("load remote JWKS: %v", err)
	}
	trustManifest := map[string]any{
		"identity":     "https://example.com/security",
		"identityType": "https",
	}
	trustManifest["signature"] = testDetachedJWS(t, "remote-ed25519", trustManifest, privateKey)
	_, err = VerifySignatures(signedCatalog(trustManifest), SignatureOptions{
		TrustAnchors: anchors,
	})
	if err == nil || !strings.Contains(err.Error(), "remote JWKS host") {
		t.Fatalf("expected remote JWKS host mismatch, got %v", err)
	}
}

func TestVerifySignaturesRejectsTampering(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	trustManifest := map[string]any{
		"identity":     "https://example.com",
		"identityType": "https",
	}
	trustManifest["signature"] = testDetachedJWS(t, "acme-ed25519", trustManifest, privateKey)
	trustManifest["identityType"] = "other"

	_, err = VerifySignatures(signedCatalog(trustManifest), SignatureOptions{
		TrustAnchors: TrustAnchors{Keys: []TrustAnchorKey{
			{
				KeyID:     "acme-ed25519",
				Algorithm: "EdDSA",
				PublicKey: base64.RawURLEncoding.EncodeToString(publicKey),
			},
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "signature verification failed") {
		t.Fatalf("expected signature verification failure, got %v", err)
	}
}

func TestVerifySignaturesCanRequireSignatures(t *testing.T) {
	_, err := VerifySignatures(signedCatalog(map[string]any{
		"identity": "https://example.com",
	}), SignatureOptions{RequireSignatures: true})
	if err == nil || !strings.Contains(err.Error(), "trustManifest.signature is required") {
		t.Fatalf("expected missing signature error, got %v", err)
	}
}

func TestVerifySignaturesRejectsUnknownKeyID(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	trustManifest := map[string]any{
		"identity": "https://example.com",
	}
	trustManifest["signature"] = testDetachedJWS(t, "missing-ed25519", trustManifest, privateKey)

	_, err = VerifySignatures(signedCatalog(trustManifest), SignatureOptions{
		TrustAnchors: TrustAnchors{Keys: []TrustAnchorKey{
			{
				KeyID:     "acme-ed25519",
				Algorithm: "EdDSA",
				PublicKey: base64.RawURLEncoding.EncodeToString(publicKey),
			},
		}},
	})
	if err == nil || !strings.Contains(err.Error(), `no trust anchor found for kid "missing-ed25519"`) {
		t.Fatalf("expected unknown key error, got %v", err)
	}
}

func mustURLHost(t *testing.T, rawURL string) string {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}
	return parsed.Hostname()
}

func mustURLAuthority(t *testing.T, rawURL string) string {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}
	return parsed.Host
}

func signedCatalog(trustManifest map[string]any) ard.Catalog {
	return ard.Catalog{
		SpecVersion: "1.0",
		Entries: []ard.CatalogEntry{
			{
				Identifier:    "urn:air:example.com:agent:test",
				DisplayName:   "Test Agent",
				Type:          ard.TypeA2AAgentCard,
				URL:           "https://example.com/agent-card.json",
				TrustManifest: trustManifest,
			},
		},
	}
}

func testDetachedJWS(t *testing.T, keyID string, trustManifest map[string]any, privateKey ed25519.PrivateKey) string {
	t.Helper()
	protected, err := json.Marshal(jwsProtectedHeader{
		Algorithm: "EdDSA",
		KeyID:     keyID,
	})
	if err != nil {
		t.Fatalf("marshal protected header: %v", err)
	}
	protectedPart := base64.RawURLEncoding.EncodeToString(protected)
	payload, err := canonicalTrustManifestPayload(trustManifest)
	if err != nil {
		t.Fatalf("canonical payload: %v", err)
	}
	signingInput := []byte(protectedPart + "." + base64.RawURLEncoding.EncodeToString(payload))
	signature := ed25519.Sign(privateKey, signingInput)
	return protectedPart + ".." + base64.RawURLEncoding.EncodeToString(signature)
}
