package adapters

import (
	"context"
	"fmt"
	"strings"

	"github.com/ifuryst/ard/internal/ard"
)

func LoadSkill(ctx context.Context, source string, options Options) (ard.CatalogEntry, error) {
	artifact, err := readSource(ctx, source, "text/markdown, text/plain")
	if err != nil {
		return ard.CatalogEntry{}, err
	}
	content := string(artifact.Data)
	frontmatter := parseFrontmatter(content)

	name := firstNonEmpty(frontmatter["name"], titleFromMarkdown(content), "Agent skill")
	description := frontmatter["description"]
	identifier, err := identifierFor(source, "skill", name, options)
	if err != nil {
		return ard.CatalogEntry{}, err
	}
	entry := ard.CatalogEntry{
		Identifier:  identifier,
		DisplayName: name,
		Type:        ard.TypeAISkill,
		Description: description,
		Tags:        []string{"skill"},
		Capabilities: uniqueStrings([]string{
			name,
			slugify(name),
		}),
		Metadata: map[string]any{
			"adapter": "skill",
		},
	}
	if artifact.IsURL {
		entry.URL = source
	} else {
		entry.Data = map[string]any{
			"name":        name,
			"description": description,
			"content":     content,
		}
	}
	if err := ard.ValidateCatalogEntry(entry); err != nil {
		return ard.CatalogEntry{}, err
	}
	return entry, nil
}

func parseFrontmatter(content string) map[string]string {
	result := map[string]string{}
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	if !strings.HasPrefix(normalized, "---\n") {
		return result
	}
	rest := strings.TrimPrefix(normalized, "---\n")
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return result
	}
	for _, line := range strings.Split(rest[:end], "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if key != "" && value != "" {
			result[key] = value
		}
	}
	return result
}

func titleFromMarkdown(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func CatalogFromEntry(entry ard.CatalogEntry) ard.Catalog {
	return ard.Catalog{
		SpecVersion: "1.0",
		Entries:     []ard.CatalogEntry{entry},
	}
}

func FormatArtifactImport(entry ard.CatalogEntry, source string) string {
	return fmt.Sprintf("imported %s %s from %s", entry.Type, entry.Identifier, source)
}
