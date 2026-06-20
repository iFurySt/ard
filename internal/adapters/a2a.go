package adapters

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ifuryst/ard/internal/ard"
)

type a2aAgentCard struct {
	Name            string         `json:"name"`
	Description     string         `json:"description"`
	URL             string         `json:"url"`
	Version         string         `json:"version"`
	ProtocolVersion string         `json:"protocolVersion"`
	Skills          []a2aSkill     `json:"skills"`
	Raw             map[string]any `json:"-"`
}

type a2aSkill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

func LoadA2AAgentCard(ctx context.Context, source string, options Options) (ard.CatalogEntry, error) {
	artifact, err := readSource(ctx, source, "application/json")
	if err != nil {
		return ard.CatalogEntry{}, err
	}

	var raw map[string]any
	if err := json.Unmarshal(artifact.Data, &raw); err != nil {
		return ard.CatalogEntry{}, fmt.Errorf("parse A2A agent card: %w", err)
	}
	var card a2aAgentCard
	if err := json.Unmarshal(artifact.Data, &card); err != nil {
		return ard.CatalogEntry{}, fmt.Errorf("parse A2A agent card: %w", err)
	}
	card.Raw = raw

	displayName := firstNonEmpty(card.Name, "A2A agent")
	identifier, err := identifierFor(source, "agent", displayName, options)
	if err != nil {
		return ard.CatalogEntry{}, err
	}
	entry := ard.CatalogEntry{
		Identifier:   identifier,
		DisplayName:  displayName,
		Type:         ard.TypeA2AAgentCard,
		Description:  card.Description,
		Version:      card.Version,
		Tags:         a2aTags(card),
		Capabilities: a2aCapabilities(card),
		Metadata: map[string]any{
			"adapter": "a2a",
		},
	}
	if card.URL != "" {
		entry.Metadata["agentUrl"] = card.URL
	}
	if card.ProtocolVersion != "" {
		entry.Metadata["protocolVersion"] = card.ProtocolVersion
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

func a2aCapabilities(card a2aAgentCard) []string {
	values := make([]string, 0, len(card.Skills)*2)
	for _, skill := range card.Skills {
		values = append(values, firstNonEmpty(skill.ID, skill.Name))
		if skill.Name != "" && skill.Name != skill.ID {
			values = append(values, skill.Name)
		}
	}
	return uniqueStrings(values)
}

func a2aTags(card a2aAgentCard) []string {
	values := []string{"a2a", "agent-card"}
	for _, skill := range card.Skills {
		values = append(values, skill.Tags...)
	}
	return uniqueStrings(values)
}
