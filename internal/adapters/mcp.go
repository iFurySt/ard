package adapters

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ifuryst/ard/internal/ard"
)

type mcpServerCard struct {
	Name        string         `json:"name"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Version     string         `json:"version"`
	WebsiteURL  string         `json:"websiteUrl"`
	Remotes     []mcpRemote    `json:"remotes"`
	Tools       []mcpTool      `json:"tools"`
	Meta        map[string]any `json:"_meta"`
	Raw         map[string]any `json:"-"`
}

type mcpRemote struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type mcpTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func LoadMCPServerCard(ctx context.Context, source string, options Options) (ard.CatalogEntry, error) {
	artifact, err := readSource(ctx, source, "application/json")
	if err != nil {
		return ard.CatalogEntry{}, err
	}

	var raw map[string]any
	if err := json.Unmarshal(artifact.Data, &raw); err != nil {
		return ard.CatalogEntry{}, fmt.Errorf("parse MCP server card: %w", err)
	}
	var card mcpServerCard
	if err := json.Unmarshal(artifact.Data, &card); err != nil {
		return ard.CatalogEntry{}, fmt.Errorf("parse MCP server card: %w", err)
	}
	card.Raw = raw

	displayName := firstNonEmpty(card.Title, card.Name, "MCP server")
	identifier, err := identifierFor(source, "server", displayName, options)
	if err != nil {
		return ard.CatalogEntry{}, err
	}
	entry := ard.CatalogEntry{
		Identifier:   identifier,
		DisplayName:  displayName,
		Type:         ard.TypeMCPServerCard,
		Description:  card.Description,
		Version:      card.Version,
		Tags:         []string{"mcp", "server-card"},
		Capabilities: mcpCapabilities(card),
		Metadata: map[string]any{
			"adapter":      "mcp",
			"artifactName": card.Name,
		},
	}
	if card.WebsiteURL != "" {
		entry.Metadata["websiteUrl"] = card.WebsiteURL
	}
	if len(card.Remotes) > 0 {
		entry.Metadata["remoteCount"] = len(card.Remotes)
	}
	if artifact.IsURL {
		entry.URL = source
	} else {
		entry.Data = raw
	}
	if err := ard.ValidateCatalogEntry(entry); err != nil {
		return ard.CatalogEntry{}, err
	}
	return entry, nil
}

func mcpCapabilities(card mcpServerCard) []string {
	values := make([]string, 0, len(card.Tools)+len(card.Remotes))
	for _, tool := range card.Tools {
		values = append(values, tool.Name)
	}
	for _, remote := range card.Remotes {
		if remote.Type != "" {
			values = append(values, "remote:"+remote.Type)
		}
	}
	return uniqueStrings(values)
}
