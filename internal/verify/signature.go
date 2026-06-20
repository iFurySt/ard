package verify

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ifuryst/ard/internal/ard"
)

type SignatureResult struct {
	Identifier string `json:"identifier"`
	KeyID      string `json:"keyId,omitempty"`
	Algorithm  string `json:"algorithm"`
	Verified   bool   `json:"verified"`
}

type SignatureOptions struct {
	RequireSignatures bool
	TrustAnchors      TrustAnchors
}

type TrustAnchors struct {
	Keys []TrustAnchorKey `json:"keys"`
}

type TrustAnchorKey struct {
	KeyID     string `json:"kid"`
	Algorithm string `json:"alg"`
	PublicKey string `json:"publicKey"`
	parsedKey ed25519.PublicKey
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
	var anchors TrustAnchors
	if err := json.Unmarshal(data, &anchors); err != nil {
		return TrustAnchors{}, err
	}
	if err := anchors.prepare(); err != nil {
		return TrustAnchors{}, err
	}
	return anchors, nil
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
	verified := ed25519.Verify(key.parsedKey, signingInput, rawSignature)
	result := SignatureResult{
		Identifier: entry.Identifier,
		KeyID:      key.KeyID,
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
