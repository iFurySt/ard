package verify

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/ifuryst/ard/internal/ard"
	"github.com/ifuryst/ard/internal/requestid"
	"github.com/ifuryst/ard/internal/tracecontext"
)

const maxRemoteJWKSBytes = 256 << 10
const maxDIDWebDocumentBytes = 256 << 10
const maxOIDCMetadataBytes = 256 << 10

type SignatureResult struct {
	Identifier string `json:"identifier"`
	KeyID      string `json:"keyId,omitempty"`
	KeySource  string `json:"keySource,omitempty"`
	Algorithm  string `json:"algorithm"`
	Verified   bool   `json:"verified"`
}

type SignatureOptions struct {
	RequireSignatures bool
	TrustAnchors      TrustAnchors
}

type TLSCertificateDiscoveryOptions struct {
	SPKIPins        map[string]string
	RequireSPKIPins bool
}

type TrustAnchors struct {
	Keys []TrustAnchorKey `json:"keys"`
}

type TrustAnchorKey struct {
	KeyID      string `json:"kid"`
	Algorithm  string `json:"alg"`
	PublicKey  string `json:"publicKey"`
	sourceURL  string
	sourceHost string
	parsedKey  ed25519.PublicKey
}

type rawTrustAnchorDocument struct {
	Keys []json.RawMessage `json:"keys"`
}

type rawTrustAnchorKey struct {
	KeyID     string `json:"kid"`
	Algorithm string `json:"alg"`
	PublicKey string `json:"publicKey"`
	KeyType   string `json:"kty"`
	Curve     string `json:"crv"`
	X         string `json:"x"`
}

type didWebDocument struct {
	ID                 string                  `json:"id"`
	VerificationMethod []didWebVerificationKey `json:"verificationMethod"`
}

type didWebVerificationKey struct {
	ID           string             `json:"id"`
	Type         string             `json:"type"`
	Controller   string             `json:"controller"`
	PublicKeyJWK *rawTrustAnchorKey `json:"publicKeyJwk"`
}

type oidcProviderMetadata struct {
	Issuer  string `json:"issuer"`
	JWKSURI string `json:"jwks_uri"`
}

type trustManifestAttestation struct {
	Type string `json:"type"`
	URI  string `json:"uri"`
}

type jwsProtectedHeader struct {
	Algorithm string `json:"alg"`
	KeyID     string `json:"kid,omitempty"`
	Type      string `json:"typ,omitempty"`
}

func LoadTrustAnchors(path string) (TrustAnchors, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return TrustAnchors{}, err
	}
	anchors, err := parseTrustAnchors(data)
	if err != nil {
		return TrustAnchors{}, err
	}
	if err := anchors.prepare(); err != nil {
		return TrustAnchors{}, err
	}
	return anchors, nil
}

func LoadRemoteTrustAnchors(ctx context.Context, jwksURL string, client *http.Client) (TrustAnchors, error) {
	return loadRemoteTrustAnchors(ctx, jwksURL, client, "")
}

func loadRemoteTrustAnchors(ctx context.Context, jwksURL string, client *http.Client, sourceHost string) (TrustAnchors, error) {
	parsed, err := url.Parse(jwksURL)
	if err != nil {
		return TrustAnchors{}, err
	}
	if parsed.Scheme != "https" || parsed.Hostname() == "" {
		return TrustAnchors{}, errors.New("remote JWKS URL must be absolute HTTPS")
	}
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURL, nil)
	if err != nil {
		return TrustAnchors{}, err
	}
	request.Header.Set("Accept", "application/jwk-set+json, application/json")
	request.Header.Set("User-Agent", "ard/0.1")
	requestid.SetHeader(request.Header, ctx)
	tracecontext.SetHeader(request.Header, ctx)
	response, err := client.Do(request)
	if err != nil {
		return TrustAnchors{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return TrustAnchors{}, fmt.Errorf("remote JWKS request failed with HTTP %d", response.StatusCode)
	}
	limited := io.LimitReader(response.Body, maxRemoteJWKSBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return TrustAnchors{}, err
	}
	if len(data) > maxRemoteJWKSBytes {
		return TrustAnchors{}, fmt.Errorf("remote JWKS exceeds %d bytes", maxRemoteJWKSBytes)
	}
	anchors, err := parseTrustAnchors(data)
	if err != nil {
		return TrustAnchors{}, err
	}
	for index := range anchors.Keys {
		anchors.Keys[index].sourceURL = jwksURL
		anchors.Keys[index].sourceHost = parsed.Hostname()
		if sourceHost != "" {
			anchors.Keys[index].sourceHost = sourceHost
		}
	}
	if err := anchors.prepare(); err != nil {
		return TrustAnchors{}, err
	}
	return anchors, nil
}

func DiscoverOIDCTrustAnchors(ctx context.Context, catalog ard.Catalog, client *http.Client) (TrustAnchors, error) {
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	anchorSets := []TrustAnchors{}
	seenIssuers := map[string]struct{}{}
	for _, entry := range catalog.Entries {
		signature := trustString(entry.TrustManifest, "signature")
		if signature == "" {
			continue
		}
		identity := trustString(entry.TrustManifest, "identity")
		issuer, configurationURL, sourceHost, ok, err := oidcConfigurationURL(identity)
		if err != nil {
			return TrustAnchors{}, fmt.Errorf("%s: %w", entry.Identifier, err)
		}
		if !ok {
			continue
		}
		if _, seen := seenIssuers[issuer]; seen {
			continue
		}
		seenIssuers[issuer] = struct{}{}
		anchors, err := loadOIDCTrustAnchors(ctx, issuer, configurationURL, sourceHost, client)
		if err != nil {
			return TrustAnchors{}, fmt.Errorf("%s: %w", entry.Identifier, err)
		}
		anchorSets = append(anchorSets, anchors)
	}
	return MergeTrustAnchors(anchorSets...), nil
}

func DiscoverTLSCertificateTrustAnchors(ctx context.Context, catalog ard.Catalog, client *http.Client) (TrustAnchors, error) {
	return DiscoverTLSCertificateTrustAnchorsWithOptions(ctx, catalog, client, TLSCertificateDiscoveryOptions{})
}

func DiscoverTLSCertificateTrustAnchorsWithOptions(ctx context.Context, catalog ard.Catalog, client *http.Client, options TLSCertificateDiscoveryOptions) (TrustAnchors, error) {
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	anchors := TrustAnchors{}
	seen := map[string]struct{}{}
	for _, entry := range catalog.Entries {
		signature := trustString(entry.TrustManifest, "signature")
		if signature == "" {
			continue
		}
		identity := trustString(entry.TrustManifest, "identity")
		parsed, ok, err := httpsTrustIdentityURL(identity)
		if err != nil {
			return TrustAnchors{}, fmt.Errorf("%s: %w", entry.Identifier, err)
		}
		if !ok {
			continue
		}
		header, _, _, err := detachedCompactJWS(signature, entry.TrustManifest)
		if err != nil {
			return TrustAnchors{}, fmt.Errorf("%s: %w", entry.Identifier, err)
		}
		key, err := loadTLSCertificateTrustAnchor(ctx, parsed, header.KeyID, client, options)
		if err != nil {
			return TrustAnchors{}, fmt.Errorf("%s: %w", entry.Identifier, err)
		}
		seenKey := key.KeyID + "\x00" + key.PublicKey + "\x00" + key.sourceHost
		if _, ok := seen[seenKey]; ok {
			continue
		}
		seen[seenKey] = struct{}{}
		anchors.Keys = append(anchors.Keys, key)
	}
	if err := anchors.prepare(); err != nil {
		return TrustAnchors{}, err
	}
	return anchors, nil
}

func DiscoverSPIFFETrustAnchors(ctx context.Context, catalog ard.Catalog, client *http.Client) (TrustAnchors, error) {
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	anchorSets := []TrustAnchors{}
	seenBundles := map[string]struct{}{}
	for _, entry := range catalog.Entries {
		signature := trustString(entry.TrustManifest, "signature")
		if signature == "" {
			continue
		}
		identity := trustString(entry.TrustManifest, "identity")
		trustDomain, ok, err := spiffeTrustDomain(identity)
		if err != nil {
			return TrustAnchors{}, fmt.Errorf("%s: %w", entry.Identifier, err)
		}
		if !ok {
			continue
		}
		bundleURLs, err := spiffeBundleURLs(entry.TrustManifest, trustDomain)
		if err != nil {
			return TrustAnchors{}, fmt.Errorf("%s: %w", entry.Identifier, err)
		}
		for _, bundleURL := range bundleURLs {
			if _, seen := seenBundles[bundleURL]; seen {
				continue
			}
			seenBundles[bundleURL] = struct{}{}
			anchors, err := loadRemoteTrustAnchors(ctx, bundleURL, client, trustDomain)
			if err != nil {
				return TrustAnchors{}, fmt.Errorf("%s: load SPIFFE bundle %s: %w", entry.Identifier, bundleURL, err)
			}
			anchorSets = append(anchorSets, anchors)
		}
	}
	return MergeTrustAnchors(anchorSets...), nil
}

func loadTLSCertificateTrustAnchor(ctx context.Context, identityURL *url.URL, keyID string, client *http.Client, options TLSCertificateDiscoveryOptions) (TrustAnchorKey, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, identityURL.String(), nil)
	if err != nil {
		return TrustAnchorKey{}, err
	}
	request.Header.Set("Accept", "*/*")
	request.Header.Set("User-Agent", "ard/0.1")
	requestid.SetHeader(request.Header, ctx)
	tracecontext.SetHeader(request.Header, ctx)
	response, err := client.Do(request)
	if err != nil {
		return TrustAnchorKey{}, err
	}
	defer response.Body.Close()
	if response.TLS == nil || len(response.TLS.PeerCertificates) == 0 {
		return TrustAnchorKey{}, errors.New("TLS peer certificate is required")
	}
	leafCertificate := response.TLS.PeerCertificates[0]
	if err := verifyTLSSPKIPin(identityURL.Hostname(), leafCertificate.RawSubjectPublicKeyInfo, options); err != nil {
		return TrustAnchorKey{}, err
	}
	publicKey, ok := leafCertificate.PublicKey.(ed25519.PublicKey)
	if !ok {
		return TrustAnchorKey{}, errors.New("TLS leaf certificate public key must be Ed25519")
	}
	if strings.TrimSpace(keyID) == "" {
		keyID = "tls-cert:" + identityURL.Hostname()
	}
	return TrustAnchorKey{
		KeyID:      keyID,
		Algorithm:  "EdDSA",
		PublicKey:  base64.RawURLEncoding.EncodeToString(publicKey),
		sourceURL:  identityURL.String(),
		sourceHost: identityURL.Hostname(),
	}, nil
}

func ParseTLSSPKIPins(values []string) (map[string]string, error) {
	pins := map[string]string{}
	for index, value := range values {
		host, pin, ok := strings.Cut(value, "=")
		if !ok {
			return nil, fmt.Errorf("TLS SPKI pin %d must use host=sha256:<hex>", index)
		}
		host = strings.ToLower(strings.TrimSpace(host))
		if host == "" || strings.ContainsAny(host, "/\\") {
			return nil, fmt.Errorf("TLS SPKI pin %d host is invalid", index)
		}
		normalizedPin, err := normalizeTLSSPKIPin(pin)
		if err != nil {
			return nil, fmt.Errorf("TLS SPKI pin %d: %w", index, err)
		}
		if _, exists := pins[host]; exists {
			return nil, fmt.Errorf("duplicate TLS SPKI pin host %q", host)
		}
		pins[host] = normalizedPin
	}
	return pins, nil
}

func verifyTLSSPKIPin(host string, rawSPKI []byte, options TLSCertificateDiscoveryOptions) error {
	host = strings.ToLower(strings.TrimSpace(host))
	expectedPin, pinned := options.SPKIPins[host]
	if !pinned {
		if options.RequireSPKIPins {
			return fmt.Errorf("TLS SPKI pin is required for host %q", host)
		}
		return nil
	}
	actualPin := tlsSPKIPin(rawSPKI)
	if actualPin != expectedPin {
		return fmt.Errorf("TLS SPKI pin mismatch for host %q", host)
	}
	return nil
}

func normalizeTLSSPKIPin(pin string) (string, error) {
	pin = strings.ToLower(strings.TrimSpace(pin))
	const prefix = "sha256:"
	if !strings.HasPrefix(pin, prefix) {
		return "", errors.New("pin must start with sha256:")
	}
	decoded, err := hex.DecodeString(strings.TrimPrefix(pin, prefix))
	if err != nil {
		return "", fmt.Errorf("pin digest must be hex: %w", err)
	}
	if len(decoded) != sha256.Size {
		return "", fmt.Errorf("pin digest must be %d bytes", sha256.Size)
	}
	return prefix + hex.EncodeToString(decoded), nil
}

func tlsSPKIPin(rawSPKI []byte) string {
	sum := sha256.Sum256(rawSPKI)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func loadOIDCTrustAnchors(ctx context.Context, issuer string, configurationURL string, sourceHost string, client *http.Client) (TrustAnchors, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, configurationURL, nil)
	if err != nil {
		return TrustAnchors{}, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "ard/0.1")
	requestid.SetHeader(request.Header, ctx)
	tracecontext.SetHeader(request.Header, ctx)
	response, err := client.Do(request)
	if err != nil {
		return TrustAnchors{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return TrustAnchors{}, fmt.Errorf("OIDC configuration request failed with HTTP %d", response.StatusCode)
	}
	limited := io.LimitReader(response.Body, maxOIDCMetadataBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return TrustAnchors{}, err
	}
	if len(data) > maxOIDCMetadataBytes {
		return TrustAnchors{}, fmt.Errorf("OIDC configuration exceeds %d bytes", maxOIDCMetadataBytes)
	}
	var metadata oidcProviderMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return TrustAnchors{}, fmt.Errorf("OIDC configuration: %w", err)
	}
	if metadata.Issuer != issuer {
		return TrustAnchors{}, fmt.Errorf("OIDC issuer %q must match trustManifest.identity %q", metadata.Issuer, issuer)
	}
	if strings.TrimSpace(metadata.JWKSURI) == "" {
		return TrustAnchors{}, errors.New("OIDC jwks_uri is required")
	}
	anchors, err := loadRemoteTrustAnchors(ctx, metadata.JWKSURI, client, sourceHost)
	if err != nil {
		return TrustAnchors{}, fmt.Errorf("load OIDC jwks_uri %s: %w", metadata.JWKSURI, err)
	}
	return anchors, nil
}

func httpsTrustIdentityURL(identity string) (*url.URL, bool, error) {
	parsed, err := url.Parse(strings.TrimSpace(identity))
	if err != nil {
		return nil, false, err
	}
	if parsed.Scheme != "https" {
		return nil, false, nil
	}
	if parsed.Hostname() == "" {
		return nil, true, errors.New("HTTPS trustManifest.identity must include a host")
	}
	if parsed.Fragment != "" {
		return nil, true, errors.New("HTTPS trustManifest.identity must not include a fragment")
	}
	return parsed, true, nil
}

func spiffeTrustDomain(identity string) (string, bool, error) {
	parsed, err := url.Parse(strings.TrimSpace(identity))
	if err != nil {
		return "", false, err
	}
	if parsed.Scheme != "spiffe" {
		return "", false, nil
	}
	if parsed.Hostname() == "" {
		return "", true, errors.New("SPIFFE trustManifest.identity must include a trust domain")
	}
	return parsed.Hostname(), true, nil
}

func spiffeBundleURLs(trustManifest map[string]any, trustDomain string) ([]string, error) {
	rawAttestations, ok := trustManifest["attestations"]
	if !ok {
		return nil, errors.New(`SPIFFE trustManifest.attestations[] with type "SPIFFE-X509" and uri is required`)
	}
	data, err := json.Marshal(rawAttestations)
	if err != nil {
		return nil, err
	}
	var attestations []trustManifestAttestation
	if err := json.Unmarshal(data, &attestations); err != nil {
		return nil, fmt.Errorf("trustManifest.attestations: %w", err)
	}
	bundleURLs := []string{}
	for index, attestation := range attestations {
		if !strings.EqualFold(strings.TrimSpace(attestation.Type), "SPIFFE-X509") {
			continue
		}
		if strings.TrimSpace(attestation.URI) == "" {
			return nil, fmt.Errorf("trustManifest.attestations[%d].uri is required for SPIFFE-X509", index)
		}
		parsed, err := url.Parse(attestation.URI)
		if err != nil {
			return nil, fmt.Errorf("trustManifest.attestations[%d].uri: %w", index, err)
		}
		if parsed.Scheme != "https" || parsed.Hostname() == "" {
			return nil, fmt.Errorf("trustManifest.attestations[%d].uri must be absolute HTTPS for SPIFFE-X509", index)
		}
		if !strings.EqualFold(parsed.Hostname(), trustDomain) {
			return nil, fmt.Errorf("trustManifest.attestations[%d].uri host %q must match SPIFFE trust domain %q", index, parsed.Hostname(), trustDomain)
		}
		bundleURLs = append(bundleURLs, parsed.String())
	}
	if len(bundleURLs) == 0 {
		return nil, errors.New(`SPIFFE trustManifest.attestations[] with type "SPIFFE-X509" and uri is required`)
	}
	return bundleURLs, nil
}

func DiscoverDIDWebTrustAnchors(ctx context.Context, catalog ard.Catalog, client *http.Client) (TrustAnchors, error) {
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	anchorSets := []TrustAnchors{}
	seenDocuments := map[string]struct{}{}
	for _, entry := range catalog.Entries {
		signature := trustString(entry.TrustManifest, "signature")
		if signature == "" {
			continue
		}
		identity := trustString(entry.TrustManifest, "identity")
		documentURL, host, ok, err := didWebDocumentURL(identity)
		if err != nil {
			return TrustAnchors{}, fmt.Errorf("%s: %w", entry.Identifier, err)
		}
		if !ok {
			continue
		}
		if _, seen := seenDocuments[documentURL]; seen {
			continue
		}
		seenDocuments[documentURL] = struct{}{}
		anchors, err := loadDIDWebDocumentTrustAnchors(ctx, identity, documentURL, host, client)
		if err != nil {
			return TrustAnchors{}, fmt.Errorf("%s: %w", entry.Identifier, err)
		}
		anchorSets = append(anchorSets, anchors)
	}
	return MergeTrustAnchors(anchorSets...), nil
}

func loadDIDWebDocumentTrustAnchors(ctx context.Context, identity string, documentURL string, host string, client *http.Client) (TrustAnchors, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, documentURL, nil)
	if err != nil {
		return TrustAnchors{}, err
	}
	request.Header.Set("Accept", "application/did+json, application/json")
	request.Header.Set("User-Agent", "ard/0.1")
	requestid.SetHeader(request.Header, ctx)
	tracecontext.SetHeader(request.Header, ctx)
	response, err := client.Do(request)
	if err != nil {
		return TrustAnchors{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return TrustAnchors{}, fmt.Errorf("did:web document request failed with HTTP %d", response.StatusCode)
	}
	limited := io.LimitReader(response.Body, maxDIDWebDocumentBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return TrustAnchors{}, err
	}
	if len(data) > maxDIDWebDocumentBytes {
		return TrustAnchors{}, fmt.Errorf("did:web document exceeds %d bytes", maxDIDWebDocumentBytes)
	}
	var document didWebDocument
	if err := json.Unmarshal(data, &document); err != nil {
		return TrustAnchors{}, fmt.Errorf("did:web document: %w", err)
	}
	if strings.TrimSpace(document.ID) != "" && document.ID != identity {
		return TrustAnchors{}, fmt.Errorf("did:web document id %q must match trustManifest.identity %q", document.ID, identity)
	}
	anchors := TrustAnchors{Keys: make([]TrustAnchorKey, 0, len(document.VerificationMethod))}
	for index, method := range document.VerificationMethod {
		if method.PublicKeyJWK == nil {
			continue
		}
		if strings.TrimSpace(method.ID) == "" {
			return TrustAnchors{}, fmt.Errorf("did:web verificationMethod[%d].id is required", index)
		}
		if method.Controller != "" && method.Controller != identity {
			return TrustAnchors{}, fmt.Errorf("did:web verificationMethod[%d].controller %q must match trustManifest.identity %q", index, method.Controller, identity)
		}
		rawKey, err := json.Marshal(method.PublicKeyJWK)
		if err != nil {
			return TrustAnchors{}, err
		}
		key, err := parseTrustAnchorKey(rawKey)
		if err != nil {
			return TrustAnchors{}, fmt.Errorf("did:web verificationMethod[%d].publicKeyJwk: %w", index, err)
		}
		if strings.TrimSpace(key.KeyID) == "" {
			key.KeyID = method.ID
		}
		key.sourceURL = documentURL
		key.sourceHost = host
		anchors.Keys = append(anchors.Keys, key)
	}
	if len(anchors.Keys) == 0 {
		return TrustAnchors{}, errors.New("did:web document has no supported verificationMethod publicKeyJwk OKP/Ed25519 keys")
	}
	if err := anchors.prepare(); err != nil {
		return TrustAnchors{}, err
	}
	return anchors, nil
}

func MergeTrustAnchors(anchorSets ...TrustAnchors) TrustAnchors {
	total := 0
	for _, anchors := range anchorSets {
		total += len(anchors.Keys)
	}
	merged := TrustAnchors{Keys: make([]TrustAnchorKey, 0, total)}
	for _, anchors := range anchorSets {
		merged.Keys = append(merged.Keys, anchors.Keys...)
	}
	return merged
}

func oidcConfigurationURL(identity string) (string, string, string, bool, error) {
	issuer := strings.TrimRight(strings.TrimSpace(identity), "/")
	if issuer == "" {
		return "", "", "", false, nil
	}
	parsed, err := url.Parse(issuer)
	if err != nil {
		return "", "", "", false, err
	}
	if parsed.Scheme != "https" {
		return "", "", "", false, nil
	}
	if parsed.Hostname() == "" {
		return "", "", "", true, errors.New("OIDC issuer must include a host")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", "", "", true, errors.New("OIDC issuer must not include query or fragment")
	}
	configuration := &url.URL{Scheme: parsed.Scheme, Host: parsed.Host}
	configuration.Path = strings.TrimRight(parsed.EscapedPath(), "/") + "/.well-known/openid-configuration"
	return issuer, configuration.String(), parsed.Hostname(), true, nil
}

func didWebDocumentURL(identity string) (string, string, bool, error) {
	method, methodID, ok := didParts(identity)
	if !ok || method != "web" {
		return "", "", false, nil
	}
	parts := strings.Split(methodID, ":")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		return "", "", true, errors.New("did:web identity must include a domain")
	}
	host, err := didWebUnescapeSegment(parts[0])
	if err != nil {
		return "", "", true, fmt.Errorf("did:web domain: %w", err)
	}
	if host == "" || strings.Contains(host, "/") || strings.Contains(host, "\\") {
		return "", "", true, errors.New("did:web domain is invalid")
	}
	parsed := &url.URL{Scheme: "https", Host: host}
	if len(parts) == 1 {
		parsed.Path = "/.well-known/did.json"
		return parsed.String(), host, true, nil
	}
	pathParts := make([]string, 0, len(parts))
	for index, part := range parts[1:] {
		segment, err := didWebUnescapeSegment(part)
		if err != nil {
			return "", "", true, fmt.Errorf("did:web path segment %d: %w", index, err)
		}
		if segment == "" || segment == "." || segment == ".." || strings.Contains(segment, "/") || strings.Contains(segment, "\\") {
			return "", "", true, fmt.Errorf("did:web path segment %d is invalid", index)
		}
		pathParts = append(pathParts, url.PathEscape(segment))
	}
	parsed.Path = "/" + strings.Join(pathParts, "/") + "/did.json"
	return parsed.String(), host, true, nil
}

func didWebUnescapeSegment(segment string) (string, error) {
	unescaped, err := url.PathUnescape(segment)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(unescaped), nil
}

func parseTrustAnchors(data []byte) (TrustAnchors, error) {
	var rawDocument rawTrustAnchorDocument
	if err := json.Unmarshal(data, &rawDocument); err != nil {
		return TrustAnchors{}, err
	}
	anchors := TrustAnchors{Keys: make([]TrustAnchorKey, 0, len(rawDocument.Keys))}
	for index, rawKey := range rawDocument.Keys {
		key, err := parseTrustAnchorKey(rawKey)
		if err != nil {
			return TrustAnchors{}, fmt.Errorf("trust anchors keys[%d]: %w", index, err)
		}
		anchors.Keys = append(anchors.Keys, key)
	}
	return anchors, nil
}

func parseTrustAnchorKey(data []byte) (TrustAnchorKey, error) {
	var rawKey rawTrustAnchorKey
	if err := json.Unmarshal(data, &rawKey); err != nil {
		return TrustAnchorKey{}, err
	}
	if rawKey.PublicKey != "" {
		return TrustAnchorKey{
			KeyID:     rawKey.KeyID,
			Algorithm: rawKey.Algorithm,
			PublicKey: rawKey.PublicKey,
		}, nil
	}
	if rawKey.KeyType == "" && rawKey.Curve == "" && rawKey.X == "" {
		return TrustAnchorKey{}, errors.New("publicKey or JWKS kty/crv/x is required")
	}
	if rawKey.KeyType != "OKP" {
		return TrustAnchorKey{}, errors.New("JWKS kty must be OKP")
	}
	if rawKey.Curve != "Ed25519" {
		return TrustAnchorKey{}, errors.New("JWKS crv must be Ed25519")
	}
	if strings.TrimSpace(rawKey.X) == "" {
		return TrustAnchorKey{}, errors.New("JWKS x is required")
	}
	return TrustAnchorKey{
		KeyID:     rawKey.KeyID,
		Algorithm: rawKey.Algorithm,
		PublicKey: rawKey.X,
	}, nil
}

func VerifySignatures(catalog ard.Catalog, options SignatureOptions) ([]SignatureResult, error) {
	if err := options.TrustAnchors.prepare(); err != nil {
		return nil, err
	}
	results := []SignatureResult{}
	for _, entry := range catalog.Entries {
		signature := trustString(entry.TrustManifest, "signature")
		if signature == "" {
			if options.RequireSignatures {
				return results, fmt.Errorf("%s: trustManifest.signature is required", entry.Identifier)
			}
			continue
		}
		if len(options.TrustAnchors.Keys) == 0 {
			return results, fmt.Errorf("%s: JWS trust anchors are required to verify trustManifest.signature", entry.Identifier)
		}
		result, err := verifyEntrySignature(entry, options.TrustAnchors)
		if err != nil {
			return results, err
		}
		results = append(results, result)
	}
	return results, nil
}

func (anchors *TrustAnchors) prepare() error {
	if anchors == nil {
		return nil
	}
	seen := map[string]struct{}{}
	for index := range anchors.Keys {
		key := &anchors.Keys[index]
		if strings.TrimSpace(key.Algorithm) == "" {
			key.Algorithm = "EdDSA"
		}
		if key.Algorithm != "EdDSA" {
			return fmt.Errorf("trust anchors keys[%d].alg must be EdDSA", index)
		}
		if strings.TrimSpace(key.KeyID) == "" {
			return fmt.Errorf("trust anchors keys[%d].kid is required", index)
		}
		if _, ok := seen[key.KeyID]; ok {
			return fmt.Errorf("duplicate trust anchor kid %q", key.KeyID)
		}
		seen[key.KeyID] = struct{}{}
		publicKey, err := decodeBase64URL(key.PublicKey)
		if err != nil {
			return fmt.Errorf("trust anchors keys[%d].publicKey: %w", index, err)
		}
		if len(publicKey) != ed25519.PublicKeySize {
			return fmt.Errorf("trust anchors keys[%d].publicKey must decode to %d bytes", index, ed25519.PublicKeySize)
		}
		key.parsedKey = ed25519.PublicKey(publicKey)
	}
	return nil
}

func verifyEntrySignature(entry ard.CatalogEntry, anchors TrustAnchors) (SignatureResult, error) {
	signature := trustString(entry.TrustManifest, "signature")
	header, signingInput, rawSignature, err := detachedCompactJWS(signature, entry.TrustManifest)
	if err != nil {
		return SignatureResult{}, fmt.Errorf("%s: %w", entry.Identifier, err)
	}
	if header.Algorithm != "EdDSA" {
		return SignatureResult{}, fmt.Errorf("%s: trustManifest.signature alg must be EdDSA", entry.Identifier)
	}
	key, err := anchors.key(header.KeyID)
	if err != nil {
		return SignatureResult{}, fmt.Errorf("%s: %w", entry.Identifier, err)
	}
	if key.sourceHost != "" {
		if err := requireRemoteKeyTrustDomain(entry, key); err != nil {
			return SignatureResult{}, err
		}
	}
	verified := ed25519.Verify(key.parsedKey, signingInput, rawSignature)
	result := SignatureResult{
		Identifier: entry.Identifier,
		KeyID:      key.KeyID,
		KeySource:  key.sourceURL,
		Algorithm:  header.Algorithm,
		Verified:   verified,
	}
	if !verified {
		return result, fmt.Errorf("%s: trustManifest.signature verification failed", entry.Identifier)
	}
	return result, nil
}

func (anchors TrustAnchors) key(keyID string) (TrustAnchorKey, error) {
	if keyID == "" {
		if len(anchors.Keys) == 1 {
			return anchors.Keys[0], nil
		}
		return TrustAnchorKey{}, errors.New("trustManifest.signature kid is required when multiple trust anchors are configured")
	}
	for _, key := range anchors.Keys {
		if key.KeyID == keyID {
			return key, nil
		}
	}
	return TrustAnchorKey{}, fmt.Errorf("no trust anchor found for kid %q", keyID)
}

func requireRemoteKeyTrustDomain(entry ard.CatalogEntry, key TrustAnchorKey) error {
	identity := trustString(entry.TrustManifest, "identity")
	trustDomain, ok := trustIdentityDomain(identity)
	if !ok {
		return fmt.Errorf("%s: remote JWKS key %q requires HTTP(S), SPIFFE, or did:web trustManifest.identity", entry.Identifier, key.KeyID)
	}
	if !strings.EqualFold(trustDomain, key.sourceHost) {
		return fmt.Errorf("%s: remote JWKS host %q must match trustManifest.identity trust domain %q", entry.Identifier, key.sourceHost, trustDomain)
	}
	return nil
}

func trustIdentityDomain(identity string) (string, bool) {
	parsed, err := url.Parse(identity)
	if err == nil {
		switch parsed.Scheme {
		case "http", "https", "spiffe":
			if parsed.Hostname() != "" {
				return parsed.Hostname(), true
			}
		case "did":
			method, methodID, ok := didParts(identity)
			if ok && method == "web" {
				domain, _, _ := strings.Cut(methodID, ":")
				if unescaped, err := url.PathUnescape(domain); err == nil && unescaped != "" {
					return unescaped, true
				}
				if domain != "" {
					return domain, true
				}
			}
		}
	}
	return "", false
}

func didParts(identity string) (string, string, bool) {
	parts := strings.SplitN(identity, ":", 3)
	if len(parts) != 3 || parts[0] != "did" || parts[1] == "" || parts[2] == "" {
		return "", "", false
	}
	return parts[1], parts[2], true
}

func detachedCompactJWS(signature string, trustManifest map[string]any) (jwsProtectedHeader, []byte, []byte, error) {
	parts := strings.Split(signature, ".")
	if len(parts) != 3 || parts[1] != "" {
		return jwsProtectedHeader{}, nil, nil, errors.New("trustManifest.signature must be detached compact JWS")
	}
	protectedBytes, err := decodeBase64URL(parts[0])
	if err != nil {
		return jwsProtectedHeader{}, nil, nil, fmt.Errorf("trustManifest.signature protected header: %w", err)
	}
	var header jwsProtectedHeader
	if err := json.Unmarshal(protectedBytes, &header); err != nil {
		return jwsProtectedHeader{}, nil, nil, fmt.Errorf("trustManifest.signature protected header: %w", err)
	}
	if strings.TrimSpace(header.Algorithm) == "" {
		return jwsProtectedHeader{}, nil, nil, errors.New("trustManifest.signature protected header alg is required")
	}
	payload, err := canonicalTrustManifestPayload(trustManifest)
	if err != nil {
		return jwsProtectedHeader{}, nil, nil, err
	}
	rawSignature, err := decodeBase64URL(parts[2])
	if err != nil {
		return jwsProtectedHeader{}, nil, nil, fmt.Errorf("trustManifest.signature value: %w", err)
	}
	signingInput := []byte(parts[0] + "." + base64.RawURLEncoding.EncodeToString(payload))
	return header, signingInput, rawSignature, nil
}

func canonicalTrustManifestPayload(trustManifest map[string]any) ([]byte, error) {
	if trustManifest == nil {
		return nil, errors.New("trustManifest is required for signature verification")
	}
	payload := make(map[string]any, len(trustManifest))
	for key, value := range trustManifest {
		if key == "signature" {
			continue
		}
		payload[key] = value
	}
	return json.Marshal(payload)
}

func decodeBase64URL(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, errors.New("value is empty")
	}
	return base64.RawURLEncoding.DecodeString(value)
}
