package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ifuryst/ard/internal/ard"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Store struct {
	db *gorm.DB
}

type CatalogEntryRecord struct {
	Identifier            string         `gorm:"primaryKey;size:512"`
	DisplayName           string         `gorm:"not null"`
	Type                  string         `gorm:"not null;index"`
	URL                   string         `gorm:""`
	Data                  datatypes.JSON `gorm:"type:jsonb"`
	Description           string         `gorm:"type:text"`
	Tags                  datatypes.JSON `gorm:"type:jsonb"`
	Capabilities          datatypes.JSON `gorm:"type:jsonb"`
	RepresentativeQueries datatypes.JSON `gorm:"type:jsonb"`
	Version               string
	UpdatedAtValue        string
	Metadata              datatypes.JSON `gorm:"type:jsonb"`
	TrustManifest         datatypes.JSON `gorm:"type:jsonb"`
	Source                string         `gorm:"not null"`
	SearchText            string         `gorm:"type:text;index"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type SearchOptions struct {
	Text     string
	Filter   ard.Filter
	Limit    int
	Source   string
	PageSize int
}

func Open(databaseURL string) (*Store, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func (store *Store) AutoMigrate() error {
	return store.db.AutoMigrate(&CatalogEntryRecord{})
}

func (store *Store) Close() error {
	db, err := store.db.DB()
	if err != nil {
		return err
	}
	return db.Close()
}

func (store *Store) UpsertCatalog(ctx context.Context, catalog ard.Catalog, source string) error {
	records := make([]CatalogEntryRecord, 0, len(catalog.Entries))
	for _, entry := range catalog.Entries {
		record, err := recordFromEntry(entry, source)
		if err != nil {
			return err
		}
		records = append(records, record)
	}
	return store.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "identifier"}},
		UpdateAll: true,
	}).Create(&records).Error
}

func (store *Store) Search(ctx context.Context, request ard.SearchRequest, source string) ([]ard.SearchResult, error) {
	limit := request.NormalizedPageSize()
	query := store.db.WithContext(ctx).Model(&CatalogEntryRecord{}).Order("display_name ASC")
	if source != "" {
		query = query.Where("source = ?", source)
	}
	if request.Query.Text != "" {
		terms := strings.Fields(strings.ToLower(request.Query.Text))
		if len(terms) > 0 {
			conditions := make([]string, 0, len(terms))
			values := make([]any, 0, len(terms))
			for _, term := range terms {
				conditions = append(conditions, "search_text ILIKE ?")
				values = append(values, "%"+term+"%")
			}
			query = query.Where(strings.Join(conditions, " OR "), values...)
		}
	}
	if types := request.Query.Filter["type"]; len(types) > 0 {
		query = query.Where("type IN ?", types)
	}

	var records []CatalogEntryRecord
	if err := query.Limit(limit * 3).Find(&records).Error; err != nil {
		return nil, err
	}

	results := make([]ard.SearchResult, 0, len(records))
	for _, record := range records {
		entry, err := record.ToCatalogEntry()
		if err != nil {
			return nil, err
		}
		if !matchesFilter(entry, request.Query.Filter) {
			continue
		}
		results = append(results, ard.SearchResult{
			CatalogEntry: entry,
			Score:        relevanceScore(entry, request.Query.Text),
			Source:       record.Source,
		})
		if len(results) >= limit {
			break
		}
	}
	return results, nil
}

func (store *Store) Count(ctx context.Context) (int64, error) {
	var count int64
	err := store.db.WithContext(ctx).Model(&CatalogEntryRecord{}).Count(&count).Error
	return count, err
}

func recordFromEntry(entry ard.CatalogEntry, source string) (CatalogEntryRecord, error) {
	return CatalogEntryRecord{
		Identifier:            entry.Identifier,
		DisplayName:           entry.DisplayName,
		Type:                  entry.Type,
		URL:                   entry.URL,
		Data:                  jsonMap(entry.Data),
		Description:           entry.Description,
		Tags:                  jsonSlice(entry.Tags),
		Capabilities:          jsonSlice(entry.Capabilities),
		RepresentativeQueries: jsonSlice(entry.RepresentativeQueries),
		Version:               entry.Version,
		UpdatedAtValue:        entry.UpdatedAt,
		Metadata:              jsonMap(entry.Metadata),
		TrustManifest:         jsonMap(entry.TrustManifest),
		Source:                source,
		SearchText:            searchText(entry),
	}, nil
}

func (record CatalogEntryRecord) ToCatalogEntry() (ard.CatalogEntry, error) {
	entry := ard.CatalogEntry{
		Identifier:  record.Identifier,
		DisplayName: record.DisplayName,
		Type:        record.Type,
		URL:         record.URL,
		Description: record.Description,
		Version:     record.Version,
		UpdatedAt:   record.UpdatedAtValue,
	}
	if len(record.Data) > 0 {
		if err := json.Unmarshal(record.Data, &entry.Data); err != nil {
			return ard.CatalogEntry{}, err
		}
	}
	if err := unmarshalJSON(record.Tags, &entry.Tags); err != nil {
		return ard.CatalogEntry{}, err
	}
	if err := unmarshalJSON(record.Capabilities, &entry.Capabilities); err != nil {
		return ard.CatalogEntry{}, err
	}
	if err := unmarshalJSON(record.RepresentativeQueries, &entry.RepresentativeQueries); err != nil {
		return ard.CatalogEntry{}, err
	}
	if err := unmarshalJSON(record.Metadata, &entry.Metadata); err != nil {
		return ard.CatalogEntry{}, err
	}
	if err := unmarshalJSON(record.TrustManifest, &entry.TrustManifest); err != nil {
		return ard.CatalogEntry{}, err
	}
	return entry, nil
}

func jsonSlice(values []string) datatypes.JSON {
	if len(values) == 0 {
		return nil
	}
	data, _ := json.Marshal(values)
	return data
}

func jsonMap(value map[string]any) datatypes.JSON {
	if value == nil {
		return nil
	}
	data, _ := json.Marshal(value)
	return data
}

func unmarshalJSON[T any](data datatypes.JSON, target *T) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, target)
}

func searchText(entry ard.CatalogEntry) string {
	parts := []string{
		entry.Identifier,
		ard.Publisher(entry.Identifier),
		entry.DisplayName,
		entry.Type,
		entry.Description,
		strings.Join(entry.Tags, " "),
		strings.Join(entry.Capabilities, " "),
		strings.Join(entry.RepresentativeQueries, " "),
	}
	return strings.ToLower(strings.Join(parts, " "))
}

func matchesFilter(entry ard.CatalogEntry, filter ard.Filter) bool {
	for field, values := range filter {
		if len(values) == 0 {
			continue
		}
		if !matchesFilterField(entry, field, values) {
			return false
		}
	}
	return true
}

func matchesFilterField(entry ard.CatalogEntry, field string, values []string) bool {
	switch field {
	case "type":
		return contains(values, entry.Type)
	case "publisher":
		return contains(values, ard.Publisher(entry.Identifier))
	case "tags":
		return intersects(values, entry.Tags)
	case "capabilities":
		return intersects(values, entry.Capabilities)
	case "identifier":
		return contains(values, entry.Identifier)
	default:
		if strings.HasPrefix(field, "metadata.") {
			key := strings.TrimPrefix(field, "metadata.")
			if value, ok := entry.Metadata[key].(string); ok {
				return contains(values, value)
			}
		}
		return false
	}
}

func relevanceScore(entry ard.CatalogEntry, text string) int {
	if strings.TrimSpace(text) == "" {
		return 50
	}
	haystack := searchText(entry)
	terms := strings.Fields(strings.ToLower(text))
	if len(terms) == 0 {
		return 50
	}
	matches := 0
	for _, term := range terms {
		if strings.Contains(haystack, term) {
			matches++
		}
	}
	if matches == 0 {
		return 0
	}
	return 50 + (50 * matches / len(terms))
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func intersects(expected []string, actual []string) bool {
	for _, value := range actual {
		if contains(expected, value) {
			return true
		}
	}
	return false
}

func FormatCatalogImport(count int, source string) string {
	return fmt.Sprintf("imported %d catalog entries from %s", count, source)
}
