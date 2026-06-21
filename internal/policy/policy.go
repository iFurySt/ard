package policy

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ifuryst/ard/internal/ard"
	"github.com/ifuryst/ard/internal/store"
)

type Policy struct {
	Version                            string   `json:"version"`
	DefaultStatus                      string   `json:"defaultStatus,omitempty"`
	DenyPublishers                     []string `json:"denyPublishers,omitempty"`
	PendingPublishers                  []string `json:"pendingPublishers,omitempty"`
	DenyTypes                          []string `json:"denyTypes,omitempty"`
	PendingTypes                       []string `json:"pendingTypes,omitempty"`
	RequiredApprovals                  int      `json:"requiredApprovals,omitempty"`
	RequireTrustManifest               bool     `json:"requireTrustManifest,omitempty"`
	RequireSourceDigestForURLArtifacts bool     `json:"requireSourceDigestForURLArtifacts,omitempty"`
	RequireJWSSignature                bool     `json:"requireJWSSignature,omitempty"`
}

type Evaluation struct {
	Identifier string `json:"identifier"`
	Status     string `json:"status"`
	Reason     string `json:"reason"`
}

type DeniedError struct {
	Identifier string
	Reason     string
}

func (err DeniedError) Error() string {
	return fmt.Sprintf("policy denied %s: %s", err.Identifier, err.Reason)
}

func LoadFile(path string) (Policy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Policy{}, err
	}
	var policy Policy
	if err := json.Unmarshal(data, &policy); err != nil {
		return Policy{}, err
	}
	if err := policy.Validate(); err != nil {
		return Policy{}, err
	}
	return policy, nil
}

func (policy Policy) Validate() error {
	if policy.Version != "" && policy.Version != "1" {
		return fmt.Errorf("policy version must be 1")
	}
	if policy.DefaultStatus != "" {
		if _, err := store.NormalizeLifecycleStatus(policy.DefaultStatus); err != nil {
			return fmt.Errorf("defaultStatus: %w", err)
		}
	}
	if policy.RequiredApprovals < 0 {
		return fmt.Errorf("requiredApprovals must be zero or greater")
	}
	return nil
}

func (policy Policy) NormalizedRequiredApprovals() int {
	if policy.RequiredApprovals > 1 {
		return policy.RequiredApprovals
	}
	return 1
}

func (policy Policy) EvaluateCatalog(catalog ard.Catalog) (map[string]string, []Evaluation, error) {
	statuses := map[string]string{}
	evaluations := make([]Evaluation, 0, len(catalog.Entries))
	for _, entry := range catalog.Entries {
		evaluation, err := policy.EvaluateEntry(entry)
		if err != nil {
			return nil, nil, err
		}
		statuses[entry.Identifier] = evaluation.Status
		evaluations = append(evaluations, evaluation)
	}
	return statuses, evaluations, nil
}

func (policy Policy) EvaluateEntry(entry ard.CatalogEntry) (Evaluation, error) {
	publisher := ard.Publisher(entry.Identifier)
	if containsFold(policy.DenyPublishers, publisher) {
		return Evaluation{}, DeniedError{Identifier: entry.Identifier, Reason: "publisher denied"}
	}
	if containsFold(policy.DenyTypes, entry.Type) {
		return Evaluation{}, DeniedError{Identifier: entry.Identifier, Reason: "type denied"}
	}
	if policy.RequireTrustManifest && len(entry.TrustManifest) == 0 {
		return Evaluation{}, DeniedError{Identifier: entry.Identifier, Reason: "trustManifest required"}
	}
	if policy.RequireSourceDigestForURLArtifacts && entry.URL != "" && trustString(entry.TrustManifest, "sourceDigest") == "" {
		return Evaluation{}, DeniedError{Identifier: entry.Identifier, Reason: "sourceDigest required for url delivery"}
	}
	if policy.RequireJWSSignature && trustString(entry.TrustManifest, "signature") == "" {
		return Evaluation{}, DeniedError{Identifier: entry.Identifier, Reason: "trustManifest.signature required"}
	}

	status := store.LifecycleStatusActive
	reason := "default active"
	if policy.DefaultStatus != "" {
		normalized, err := store.NormalizeLifecycleStatus(policy.DefaultStatus)
		if err != nil {
			return Evaluation{}, fmt.Errorf("defaultStatus: %w", err)
		}
		status = normalized
		reason = "default " + normalized
	}
	if containsFold(policy.PendingPublishers, publisher) {
		status = store.LifecycleStatusPending
		reason = "publisher requires review"
	}
	if containsFold(policy.PendingTypes, entry.Type) {
		status = store.LifecycleStatusPending
		reason = "type requires review"
	}
	return Evaluation{Identifier: entry.Identifier, Status: status, Reason: reason}, nil
}

func trustString(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, _ := values[key].(string)
	return strings.TrimSpace(value)
}

func containsFold(values []string, target string) bool {
	target = strings.ToLower(strings.TrimSpace(target))
	for _, value := range values {
		if strings.ToLower(strings.TrimSpace(value)) == target {
			return true
		}
	}
	return false
}
