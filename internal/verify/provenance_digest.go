package verify

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/ifuryst/ard/internal/ard"
)

type ProvenanceDigestResult struct {
	Identifier string `json:"identifier"`
	Index      int    `json:"index"`
	Relation   string `json:"relation,omitempty"`
	SourceID   string `json:"sourceId"`
	Expected   string `json:"expected"`
	Actual     string `json:"actual"`
	Verified   bool   `json:"verified"`
}

type ProvenanceDigestOptions struct {
	RequirePinnedHTTPSourceIDs bool
}

func VerifyProvenanceDigests(ctx context.Context, catalog ard.Catalog) ([]ProvenanceDigestResult, error) {
	return VerifyProvenanceDigestsWithOptions(ctx, catalog, ProvenanceDigestOptions{})
}

func VerifyProvenanceDigestsWithOptions(ctx context.Context, catalog ard.Catalog, options ProvenanceDigestOptions) ([]ProvenanceDigestResult, error) {
	results := []ProvenanceDigestResult{}
	for _, entry := range catalog.Entries {
		provenance, err := trustProvenance(entry.TrustManifest)
		if err != nil {
			return results, fmt.Errorf("%s: %w", entry.Identifier, err)
		}
		for index, link := range provenance {
			sourceID := trustString(link, "sourceId")
			expected := trustString(link, "sourceDigest")
			retrievable := isHTTPSourceID(sourceID)
			if expected == "" {
				if options.RequirePinnedHTTPSourceIDs && retrievable {
					return results, fmt.Errorf("%s: trustManifest.provenance[%d].sourceDigest required for HTTP(S) sourceId", entry.Identifier, index)
				}
				continue
			}
			if !retrievable {
				return results, fmt.Errorf("%s: trustManifest.provenance[%d].sourceDigest verification requires HTTP(S) sourceId", entry.Identifier, index)
			}
			actual, err := fetchDigest(ctx, sourceID)
			if err != nil {
				return results, fmt.Errorf("%s: verify trustManifest.provenance[%d].sourceDigest: %w", entry.Identifier, index, err)
			}
			result := ProvenanceDigestResult{
				Identifier: entry.Identifier,
				Index:      index,
				Relation:   trustString(link, "relation"),
				SourceID:   sourceID,
				Expected:   expected,
				Actual:     actual,
				Verified:   strings.EqualFold(expected, actual),
			}
			results = append(results, result)
			if !result.Verified {
				return results, fmt.Errorf("%s: trustManifest.provenance[%d].sourceDigest mismatch: expected %s, got %s", entry.Identifier, index, expected, actual)
			}
		}
	}
	return results, nil
}

func trustProvenance(values map[string]any) ([]map[string]any, error) {
	if values == nil {
		return nil, nil
	}
	value, ok := values["provenance"]
	if !ok {
		return nil, nil
	}
	switch items := value.(type) {
	case []any:
		provenance := make([]map[string]any, 0, len(items))
		for index, item := range items {
			link, ok := item.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("trustManifest.provenance[%d] must be an object", index)
			}
			provenance = append(provenance, link)
		}
		return provenance, nil
	case []map[string]any:
		return items, nil
	default:
		return nil, fmt.Errorf("trustManifest.provenance must be an array")
	}
}

func isHTTPSourceID(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	return (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Hostname() != ""
}
