package verify

import (
	"context"
	"fmt"
	"strings"

	"github.com/ifuryst/ard/internal/ard"
)

type AttestationDigestResult struct {
	Identifier string `json:"identifier"`
	Index      int    `json:"index"`
	Type       string `json:"type,omitempty"`
	URI        string `json:"uri"`
	Expected   string `json:"expected"`
	Actual     string `json:"actual"`
	Verified   bool   `json:"verified"`
}

type AttestationDigestOptions struct {
	RequirePinnedAttestations bool
}

func VerifyAttestationDigests(ctx context.Context, catalog ard.Catalog) ([]AttestationDigestResult, error) {
	return VerifyAttestationDigestsWithOptions(ctx, catalog, AttestationDigestOptions{})
}

func VerifyAttestationDigestsWithOptions(ctx context.Context, catalog ard.Catalog, options AttestationDigestOptions) ([]AttestationDigestResult, error) {
	results := []AttestationDigestResult{}
	for _, entry := range catalog.Entries {
		attestations, err := trustAttestations(entry.TrustManifest)
		if err != nil {
			return results, fmt.Errorf("%s: %w", entry.Identifier, err)
		}
		for index, attestation := range attestations {
			expected := trustString(attestation, "digest")
			if expected == "" {
				if options.RequirePinnedAttestations {
					return results, fmt.Errorf("%s: trustManifest.attestations[%d].digest required", entry.Identifier, index)
				}
				continue
			}
			uri := trustString(attestation, "uri")
			if uri == "" {
				return results, fmt.Errorf("%s: trustManifest.attestations[%d].uri required for digest verification", entry.Identifier, index)
			}
			actual, err := fetchDigest(ctx, uri)
			if err != nil {
				return results, fmt.Errorf("%s: verify trustManifest.attestations[%d].digest: %w", entry.Identifier, index, err)
			}
			result := AttestationDigestResult{
				Identifier: entry.Identifier,
				Index:      index,
				Type:       trustString(attestation, "type"),
				URI:        uri,
				Expected:   expected,
				Actual:     actual,
				Verified:   strings.EqualFold(expected, actual),
			}
			results = append(results, result)
			if !result.Verified {
				return results, fmt.Errorf("%s: trustManifest.attestations[%d].digest mismatch: expected %s, got %s", entry.Identifier, index, expected, actual)
			}
		}
	}
	return results, nil
}

func trustAttestations(values map[string]any) ([]map[string]any, error) {
	if values == nil {
		return nil, nil
	}
	value, ok := values["attestations"]
	if !ok {
		return nil, nil
	}
	switch items := value.(type) {
	case []any:
		attestations := make([]map[string]any, 0, len(items))
		for index, item := range items {
			attestation, ok := item.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("trustManifest.attestations[%d] must be an object", index)
			}
			attestations = append(attestations, attestation)
		}
		return attestations, nil
	case []map[string]any:
		return items, nil
	default:
		return nil, fmt.Errorf("trustManifest.attestations must be an array")
	}
}
