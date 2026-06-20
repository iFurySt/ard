package ard

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

const (
	TypeAICatalog      = "application/ai-catalog+json"
	TypeAIRegistry     = "application/ai-registry+json"
	TypeAIRegistryBare = "application/ai-registry"
	TypeA2AAgentCard   = "application/a2a-agent-card+json"
	TypeMCPServerCard  = "application/mcp-server-card+json"
	TypeAISkill        = "text/markdown; profile=\"urn:air:agent-skills\""
)

var urnPattern = regexp.MustCompile(`^urn:air:([A-Za-z0-9.-]+)(?::([A-Za-z0-9._:-]+))?:([A-Za-z0-9._-]+)$`)

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

func ValidateCatalogEntry(entry CatalogEntry) error {
	if entry.Identifier == "" {
		return errors.New("identifier is required")
	}
	if !urnPattern.MatchString(entry.Identifier) {
		return fmt.Errorf("identifier %q must match urn:air:<publisher>:<name>", entry.Identifier)
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
		if _, err := url.ParseRequestURI(entry.URL); err != nil {
			return fmt.Errorf("url is invalid: %w", err)
		}
	}
	if queries := len(entry.RepresentativeQueries); queries > 0 && (queries < 2 || queries > 5) {
		return fmt.Errorf("representativeQueries must contain 2 to 5 items when present")
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
