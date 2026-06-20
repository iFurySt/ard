package ard

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	TypeAICatalog      = "application/ai-catalog+json"
	TypeAIRegistry     = "application/ai-registry+json"
	TypeAIRegistryBare = "application/ai-registry"
	TypeA2AAgentCard   = "application/a2a-agent-card+json"
	TypeMCPServerCard  = "application/mcp-server-card+json"
	TypeAISkill        = "text/markdown; profile=\"urn:air:agent-skills\""
	TypeOpenAPI        = "application/openapi+json"
)

var urnPattern = regexp.MustCompile(`^urn:air:([A-Za-z0-9.-]+)(?::([A-Za-z0-9._:-]+))?:([A-Za-z0-9._-]+)$`)
var sha256DigestPattern = regexp.MustCompile(`^sha256:[a-fA-F0-9]{64}$`)
var supportedTrustIdentityTypes = map[string]struct{}{
	"spiffe": {},
	"did":    {},
	"https":  {},
	"other":  {},
}
var supportedTrustProvenanceRelations = map[string]struct{}{
	"derivedFrom":   {},
	"publishedFrom": {},
	"copiedFrom":    {},
}

type Catalog struct {
	SpecVersion string         `json:"specVersion"`
	Host        *HostInfo      `json:"host,omitempty"`
	Entries     []CatalogEntry `json:"entries"`
}

type HostInfo struct {
	DisplayName      string         `json:"displayName"`
	Identifier       string         `json:"identifier,omitempty"`
	DocumentationURL string         `json:"documentationUrl,omitempty"`
	LogoURL          string         `json:"logoUrl,omitempty"`
	TrustManifest    map[string]any `json:"trustManifest,omitempty"`
}

type CatalogEntry struct {
	Identifier            string         `json:"identifier"`
	DisplayName           string         `json:"displayName"`
	Type                  string         `json:"type"`
	URL                   string         `json:"url,omitempty"`
	Data                  map[string]any `json:"data,omitempty"`
	Description           string         `json:"description,omitempty"`
	Tags                  []string       `json:"tags,omitempty"`
	Capabilities          []string       `json:"capabilities,omitempty"`
	RepresentativeQueries []string       `json:"representativeQueries,omitempty"`
	Version               string         `json:"version,omitempty"`
	UpdatedAt             string         `json:"updatedAt,omitempty"`
	Metadata              map[string]any `json:"metadata,omitempty"`
	TrustManifest         map[string]any `json:"trustManifest,omitempty"`
}

type SearchQuery struct {
	Text   string `json:"text,omitempty"`
	Filter Filter `json:"filter,omitempty"`
}

type Filter map[string][]string

type SearchRequest struct {
	Query      SearchQuery `json:"query"`
	Federation string      `json:"federation,omitempty"`
	PageSize   int         `json:"pageSize,omitempty"`
	PageToken  string      `json:"pageToken,omitempty"`
}

type SearchResult struct {
	CatalogEntry
	Score  int    `json:"score"`
	Source string `json:"source"`
}

type SearchResponse struct {
	Results   []SearchResult `json:"results"`
	Referrals []CatalogEntry `json:"referrals,omitempty"`
	PageToken string         `json:"pageToken,omitempty"`
}

type ExploreRequest struct {
	Query      SearchQuery       `json:"query,omitempty"`
	ResultType ExploreResultType `json:"resultType"`
}

type ExploreResultType struct {
	Facets []ExploreFacetRequest `json:"facets"`
}

type ExploreFacetRequest struct {
	Field    string `json:"field"`
	Limit    int    `json:"limit,omitempty"`
	MinCount int    `json:"minCount,omitempty"`
}

type ExploreResponse struct {
	ResultType string                  `json:"resultType"`
	Facets     map[string]ExploreFacet `json:"facets"`
}

type ExploreFacet struct {
	Buckets    []ExploreFacetBucket `json:"buckets"`
	OtherCount int                  `json:"otherCount,omitempty"`
}

type ExploreFacetBucket struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

type ListResponse struct {
	Items     []CatalogEntry `json:"items"`
	Total     int            `json:"total,omitempty"`
	PageToken string         `json:"pageToken,omitempty"`
}

func (filter *Filter) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		*filter = nil
		return nil
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	parsed := make(Filter, len(raw))
	for key, value := range raw {
		values, err := filterValues(value)
		if err != nil {
			return fmt.Errorf("filter %q: %w", key, err)
		}
		parsed[key] = values
	}
	*filter = parsed
	return nil
}

func (filter Filter) MarshalJSON() ([]byte, error) {
	if filter == nil {
		return []byte("{}"), nil
	}
	type alias map[string][]string
	return json.Marshal(alias(filter))
}

func (request SearchRequest) NormalizedPageSize() int {
	if request.PageSize <= 0 {
		return 10
	}
	if request.PageSize > 100 {
		return 100
	}
	return request.PageSize
}

func (request SearchRequest) NormalizedFederation() string {
	switch request.Federation {
	case "", "auto":
		return "auto"
	case "referrals", "none":
		return request.Federation
	default:
		return "auto"
	}
}

func ValidateCatalog(catalog Catalog) error {
	if catalog.SpecVersion != "1.0" {
		return fmt.Errorf("specVersion must be 1.0")
	}
	if catalog.Host != nil {
		if err := validateHostInfo(*catalog.Host); err != nil {
			return fmt.Errorf("host: %w", err)
		}
	}
	if len(catalog.Entries) == 0 {
		return errors.New("entries must not be empty")
	}
	for index, entry := range catalog.Entries {
		if err := ValidateCatalogEntry(entry); err != nil {
			return fmt.Errorf("entries[%d]: %w", index, err)
		}
	}
	return nil
}

func validateHostInfo(host HostInfo) error {
	if strings.TrimSpace(host.DisplayName) == "" {
		return errors.New("displayName is required")
	}
	if host.DocumentationURL != "" {
		if err := validateAbsoluteURI(host.DocumentationURL); err != nil {
			return fmt.Errorf("documentationUrl is invalid: %w", err)
		}
	}
	if host.LogoURL != "" {
		if err := validateAbsoluteURI(host.LogoURL); err != nil {
			return fmt.Errorf("logoUrl is invalid: %w", err)
		}
	}
	if err := validateTrustManifest("", host.TrustManifest); err != nil {
		return err
	}
	return nil
}

func ValidateCatalogEntry(entry CatalogEntry) error {
	if err := ValidateIdentifier(entry.Identifier); err != nil {
		return err
	}
	if entry.DisplayName == "" {
		return errors.New("displayName is required")
	}
	if entry.Type == "" {
		return errors.New("type is required")
	}
	if (entry.URL == "") == (entry.Data == nil) {
		return errors.New("exactly one of url or data must be present")
	}
	if entry.URL != "" {
		if err := validateHTTPURL(entry.URL); err != nil {
			return fmt.Errorf("url is invalid: %w", err)
		}
	}
	if queries := len(entry.RepresentativeQueries); queries > 0 && (queries < 2 || queries > 5) {
		return fmt.Errorf("representativeQueries must contain 2 to 5 items when present")
	}
	if entry.UpdatedAt != "" {
		if _, err := time.Parse(time.RFC3339Nano, entry.UpdatedAt); err != nil {
			return fmt.Errorf("updatedAt must be a valid RFC3339 date-time: %w", err)
		}
	}
	if err := validateMetadata(entry.Metadata); err != nil {
		return err
	}
	if err := validateTrustManifest(entry.Identifier, entry.TrustManifest); err != nil {
		return err
	}
	return nil
}

func validateMetadata(metadata map[string]any) error {
	for key, value := range metadata {
		switch value.(type) {
		case nil, string, bool,
			int, int8, int16, int32, int64,
			uint, uint8, uint16, uint32, uint64,
			float32, float64, json.Number:
			continue
		default:
			return fmt.Errorf("metadata.%s must be a string, number, boolean, or null", key)
		}
	}
	return nil
}

func validateTrustManifest(identifier string, trustManifest map[string]any) error {
	if trustManifest == nil {
		return nil
	}
	identity, ok := trustManifest["identity"].(string)
	identity = strings.TrimSpace(identity)
	if !ok || identity == "" {
		return errors.New("trustManifest.identity is required when trustManifest is present")
	}
	if parsed, err := url.Parse(identity); err == nil && parsed.Scheme != "" {
		switch parsed.Scheme {
		case "http", "https":
			if parsed.Hostname() == "" {
				return errors.New("trustManifest.identity URL must include a host")
			}
			publisher := Publisher(identifier)
			if publisher != "" && !strings.EqualFold(parsed.Hostname(), publisher) {
				return fmt.Errorf("trustManifest.identity host %q must match identifier publisher %q", parsed.Hostname(), publisher)
			}
		}
	}
	if identityType, ok := trustManifest["identityType"]; ok {
		identityTypeString, ok := identityType.(string)
		if !ok {
			return errors.New("trustManifest.identityType must be a string")
		}
		if _, ok := supportedTrustIdentityTypes[identityTypeString]; !ok {
			return errors.New("trustManifest.identityType must be one of spiffe, did, https, other")
		}
	}
	if trustSchema, ok := trustManifest["trustSchema"]; ok {
		if err := validateTrustSchema(trustSchema); err != nil {
			return err
		}
	}
	if sourceDigest, ok := trustManifest["sourceDigest"]; ok {
		sourceDigestString, ok := sourceDigest.(string)
		if !ok {
			return errors.New("trustManifest.sourceDigest must be a string")
		}
		if sourceDigestString != "" && !sha256DigestPattern.MatchString(sourceDigestString) {
			return errors.New("trustManifest.sourceDigest must match sha256:<64 hex chars>")
		}
	}
	if attestations, ok := trustManifest["attestations"]; ok {
		if err := validateTrustAttestations(attestations); err != nil {
			return err
		}
	}
	if provenance, ok := trustManifest["provenance"]; ok {
		if err := validateTrustProvenance(provenance); err != nil {
			return err
		}
	}
	if signature, ok := trustManifest["signature"]; ok {
		if _, ok := signature.(string); !ok {
			return errors.New("trustManifest.signature must be a string")
		}
	}
	return nil
}

func validateTrustSchema(value any) error {
	schema, ok := value.(map[string]any)
	if !ok {
		return errors.New("trustManifest.trustSchema must be an object")
	}
	path := "trustManifest.trustSchema"
	if _, err := requiredTrustString(schema, "identifier", path); err != nil {
		return err
	}
	if _, err := requiredTrustString(schema, "version", path); err != nil {
		return err
	}
	if governanceURI, ok, err := optionalTrustString(schema, "governanceUri", path); err != nil {
		return err
	} else if ok && governanceURI != "" {
		if err := validateAbsoluteURI(governanceURI); err != nil {
			return fmt.Errorf("%s.governanceUri is invalid: %w", path, err)
		}
	}
	if methods, ok := schema["verificationMethods"]; ok {
		if err := validateTrustStringList(methods, path+".verificationMethods"); err != nil {
			return err
		}
	}
	return nil
}

func validateTrustAttestations(value any) error {
	attestations, err := trustObjectList(value, "trustManifest.attestations")
	if err != nil {
		return err
	}
	for index, attestation := range attestations {
		path := fmt.Sprintf("trustManifest.attestations[%d]", index)
		if _, err := requiredTrustString(attestation, "type", path); err != nil {
			return err
		}
		if uri, err := requiredTrustString(attestation, "uri", path); err != nil {
			return err
		} else if err := validateAbsoluteURI(uri); err != nil {
			return fmt.Errorf("%s.uri is invalid: %w", path, err)
		}
		if _, err := requiredTrustString(attestation, "mediaType", path); err != nil {
			return err
		}
		if digest, ok, err := optionalTrustString(attestation, "digest", path); err != nil {
			return err
		} else if ok && digest != "" && !sha256DigestPattern.MatchString(digest) {
			return fmt.Errorf("%s.digest must match sha256:<64 hex chars>", path)
		}
	}
	return nil
}

func validateTrustProvenance(value any) error {
	provenance, err := trustObjectList(value, "trustManifest.provenance")
	if err != nil {
		return err
	}
	for index, link := range provenance {
		path := fmt.Sprintf("trustManifest.provenance[%d]", index)
		relation, err := requiredTrustString(link, "relation", path)
		if err != nil {
			return err
		}
		if _, ok := supportedTrustProvenanceRelations[relation]; !ok {
			return fmt.Errorf("%s.relation must be one of derivedFrom, publishedFrom, copiedFrom", path)
		}
		if _, err := requiredTrustString(link, "sourceId", path); err != nil {
			return err
		}
		if digest, ok, err := optionalTrustString(link, "sourceDigest", path); err != nil {
			return err
		} else if ok && digest != "" && !sha256DigestPattern.MatchString(digest) {
			return fmt.Errorf("%s.sourceDigest must match sha256:<64 hex chars>", path)
		}
	}
	return nil
}

func trustObjectList(value any, path string) ([]map[string]any, error) {
	switch items := value.(type) {
	case []any:
		objects := make([]map[string]any, 0, len(items))
		for index, item := range items {
			object, ok := item.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("%s[%d] must be an object", path, index)
			}
			objects = append(objects, object)
		}
		return objects, nil
	case []map[string]any:
		return items, nil
	default:
		return nil, fmt.Errorf("%s must be an array", path)
	}
}

func validateTrustStringList(value any, path string) error {
	switch items := value.(type) {
	case []any:
		for index, item := range items {
			if _, ok := item.(string); !ok {
				return fmt.Errorf("%s[%d] must be a string", path, index)
			}
		}
		return nil
	case []string:
		return nil
	default:
		return fmt.Errorf("%s must be an array", path)
	}
}

func requiredTrustString(object map[string]any, field string, path string) (string, error) {
	value, ok, err := optionalTrustString(object, field, path)
	if err != nil {
		return "", err
	}
	if !ok || value == "" {
		return "", fmt.Errorf("%s.%s is required", path, field)
	}
	return value, nil
}

func optionalTrustString(object map[string]any, field string, path string) (string, bool, error) {
	value, ok := object[field]
	if !ok {
		return "", false, nil
	}
	valueString, ok := value.(string)
	if !ok {
		return "", true, fmt.Errorf("%s.%s must be a string", path, field)
	}
	return strings.TrimSpace(valueString), true, nil
}

func validateAbsoluteURI(value string) error {
	parsed, err := url.Parse(value)
	if err != nil {
		return err
	}
	if parsed.Scheme == "" {
		return errors.New("scheme is required")
	}
	return nil
}

func validateHTTPURL(value string) error {
	parsed, err := url.Parse(value)
	if err != nil {
		return err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("scheme must be http or https")
	}
	if parsed.Host == "" {
		return errors.New("host is required")
	}
	return nil
}

func ValidateIdentifier(identifier string) error {
	if identifier == "" {
		return errors.New("identifier is required")
	}
	if !urnPattern.MatchString(identifier) {
		return fmt.Errorf("identifier %q must match urn:air:<publisher>:<name>", identifier)
	}
	return nil
}

func Publisher(identifier string) string {
	if !strings.HasPrefix(identifier, "urn:air:") {
		return ""
	}
	rest := strings.TrimPrefix(identifier, "urn:air:")
	publisher, _, _ := strings.Cut(rest, ":")
	return publisher
}

func filterValues(value any) ([]string, error) {
	switch typed := value.(type) {
	case string:
		return []string{typed}, nil
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			itemString, ok := item.(string)
			if !ok {
				return nil, errors.New("array values must be strings")
			}
			values = append(values, itemString)
		}
		return values, nil
	default:
		return nil, errors.New("value must be a string or string array")
	}
}
