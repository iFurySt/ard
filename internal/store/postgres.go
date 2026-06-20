package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ifuryst/ard/internal/ard"
	"github.com/ifuryst/ard/internal/pagination"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Store struct {
	db *gorm.DB
}

const (
	LifecycleStatusActive   = "active"
	LifecycleStatusPending  = "pending"
	LifecycleStatusDisabled = "disabled"
)

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
	LifecycleStatus       string         `gorm:"not null;default:active;index"`
	SearchText            string         `gorm:"type:text;index"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type AuditEventRecord struct {
	ID         string `gorm:"primaryKey;size:36"`
	Action     string `gorm:"not null;index"`
	Identifier string `gorm:"index"`
	Status     string
	RequestID  string `gorm:"index"`
	Source     string `gorm:"not null;index"`
	RemoteAddr string
	CreatedAt  time.Time `gorm:"index"`
}

type AuditEvent struct {
	ID         string    `json:"id"`
	Action     string    `json:"action"`
	Identifier string    `json:"identifier,omitempty"`
	Status     string    `json:"status,omitempty"`
	RequestID  string    `json:"requestId,omitempty"`
	Source     string    `json:"source"`
	RemoteAddr string    `json:"remoteAddr,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
}

type SearchOptions struct {
	Text     string
	Filter   ard.Filter
	Limit    int
	Source   string
	PageSize int
}

type SearchPage struct {
	Results       []ard.SearchResult
	NextPageToken string
}

type ListOptions struct {
	Limit                    int
	PageToken                string
	Type                     string
	Status                   string
	IncludeInactive          bool
	IncludeLifecycleMetadata bool
}

type ListEntriesPage struct {
	Entries       []ard.CatalogEntry
	Total         int64
	NextPageToken string
}

func Open(databaseURL string) (*Store, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func (store *Store) AutoMigrate() error {
	return store.db.AutoMigrate(&CatalogEntryRecord{}, &AuditEventRecord{})
}

func (store *Store) Close() error {
	db, err := store.db.DB()
	if err != nil {
		return err
	}
	return db.Close()
}

func (store *Store) UpsertCatalog(ctx context.Context, catalog ard.Catalog, source string) error {
	return store.UpsertCatalogWithStatuses(ctx, catalog, source, nil)
}

func (store *Store) UpsertCatalogWithStatuses(ctx context.Context, catalog ard.Catalog, source string, statuses map[string]string) error {
	records := make([]CatalogEntryRecord, 0, len(catalog.Entries))
	statusUpdates := map[string]string{}
	for _, entry := range catalog.Entries {
		status := LifecycleStatusActive
		if statuses != nil && statuses[entry.Identifier] != "" {
			status = statuses[entry.Identifier]
		}
		record, err := recordFromEntryWithStatus(entry, source, status)
		if err != nil {
			return err
		}
		records = append(records, record)
		if status != LifecycleStatusActive {
			statusUpdates[entry.Identifier] = status
		}
	}
	return store.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "identifier"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"display_name",
				"type",
				"url",
				"data",
				"description",
				"tags",
				"capabilities",
				"representative_queries",
				"version",
				"updated_at_value",
				"metadata",
				"trust_manifest",
				"source",
				"search_text",
				"updated_at",
			}),
		}).Create(&records).Error; err != nil {
			return err
		}
		for identifier, status := range statusUpdates {
			if err := tx.Model(&CatalogEntryRecord{}).
				Where("identifier = ?", identifier).
				Update("lifecycle_status", status).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (store *Store) Search(ctx context.Context, request ard.SearchRequest, source string) ([]ard.SearchResult, error) {
	page, err := store.SearchPage(ctx, request, source)
	if err != nil {
		return nil, err
	}
	return page.Results, nil
}

func (store *Store) SearchPage(ctx context.Context, request ard.SearchRequest, source string) (SearchPage, error) {
	limit := request.NormalizedPageSize()
	offset, err := pagination.Offset(request.PageToken)
	if err != nil {
		return SearchPage{}, err
	}
	records, err := store.matchingRecords(ctx, request.Query, source, 0)
	if err != nil {
		return SearchPage{}, err
	}

	results := make([]ard.SearchResult, 0, len(records))
	for _, record := range records {
		entry, err := record.ToCatalogEntry()
		if err != nil {
			return SearchPage{}, err
		}
		if !matchesFilter(entry, request.Query.Filter) {
			continue
		}
		results = append(results, ard.SearchResult{
			CatalogEntry: entry,
			Score:        relevanceScore(entry, request.Query.Text),
			Source:       record.Source,
		})
	}
	if offset > len(results) {
		offset = len(results)
	}
	end := offset + limit
	nextToken := ""
	if end < len(results) {
		nextToken = pagination.Token(end)
	} else {
		end = len(results)
	}
	return SearchPage{Results: results[offset:end], NextPageToken: nextToken}, nil
}

func (store *Store) RegistryReferrals(ctx context.Context, limit int) ([]ard.CatalogEntry, error) {
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	var records []CatalogEntryRecord
	if err := store.db.WithContext(ctx).
		Where("lifecycle_status = ?", LifecycleStatusActive).
		Where("type IN ?", []string{ard.TypeAIRegistry, ard.TypeAIRegistryBare}).
		Order("display_name ASC").
		Limit(limit).
		Find(&records).Error; err != nil {
		return nil, err
	}
	entries := make([]ard.CatalogEntry, 0, len(records))
	for _, record := range records {
		entry, err := record.ToCatalogEntry()
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (store *Store) List(ctx context.Context, limit int) ([]ard.CatalogEntry, int64, error) {
	return store.ListEntries(ctx, ListOptions{Limit: limit})
}

func (store *Store) ListEntries(ctx context.Context, options ListOptions) ([]ard.CatalogEntry, int64, error) {
	page, err := store.ListEntriesPage(ctx, options)
	if err != nil {
		return nil, 0, err
	}
	return page.Entries, page.Total, nil
}

func (store *Store) ListEntriesPage(ctx context.Context, options ListOptions) (ListEntriesPage, error) {
	limit := options.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset, err := pagination.Offset(options.PageToken)
	if err != nil {
		return ListEntriesPage{}, err
	}
	listQuery := func() *gorm.DB {
		query := store.db.WithContext(ctx).Model(&CatalogEntryRecord{})
		if options.Type != "" {
			query = query.Where("type = ?", options.Type)
		}
		if options.Status != "" {
			query = query.Where("lifecycle_status = ?", options.Status)
		} else if !options.IncludeInactive {
			query = query.Where("lifecycle_status = ?", LifecycleStatusActive)
		}
		return query
	}
	var total int64
	if err := listQuery().Count(&total).Error; err != nil {
		return ListEntriesPage{}, err
	}
	var records []CatalogEntryRecord
	if err := listQuery().Order("display_name ASC").Offset(offset).Limit(limit + 1).Find(&records).Error; err != nil {
		return ListEntriesPage{}, err
	}
	nextToken := ""
	if len(records) > limit {
		nextToken = pagination.Token(offset + limit)
		records = records[:limit]
	}
	entries := make([]ard.CatalogEntry, 0, len(records))
	for _, record := range records {
		entry, err := record.toCatalogEntry(options.IncludeLifecycleMetadata)
		if err != nil {
			return ListEntriesPage{}, err
		}
		entries = append(entries, entry)
	}
	return ListEntriesPage{Entries: entries, Total: total, NextPageToken: nextToken}, nil
}

func (store *Store) GetEntry(ctx context.Context, identifier string, includeLifecycleMetadata bool) (ard.CatalogEntry, bool, error) {
	var record CatalogEntryRecord
	if err := store.db.WithContext(ctx).First(&record, "identifier = ?", identifier).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ard.CatalogEntry{}, false, nil
		}
		return ard.CatalogEntry{}, false, err
	}
	entry, err := record.toCatalogEntry(includeLifecycleMetadata)
	if err != nil {
		return ard.CatalogEntry{}, false, err
	}
	return entry, true, nil
}

func (store *Store) SetEntryStatus(ctx context.Context, identifier string, status string) (bool, error) {
	normalized, err := NormalizeLifecycleStatus(status)
	if err != nil {
		return false, err
	}
	result := store.db.WithContext(ctx).Model(&CatalogEntryRecord{}).
		Where("identifier = ?", identifier).
		Update("lifecycle_status", normalized)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (store *Store) DeleteEntry(ctx context.Context, identifier string) (bool, error) {
	result := store.db.WithContext(ctx).Delete(&CatalogEntryRecord{}, "identifier = ?", identifier)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (store *Store) RecordAuditEvent(ctx context.Context, event AuditEvent) error {
	record := AuditEventRecord{
		ID:         event.ID,
		Action:     event.Action,
		Identifier: event.Identifier,
		Status:     event.Status,
		RequestID:  event.RequestID,
		Source:     event.Source,
		RemoteAddr: event.RemoteAddr,
		CreatedAt:  event.CreatedAt,
	}
	if record.ID == "" {
		record.ID = uuid.NewString()
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC()
	}
	return store.db.WithContext(ctx).Create(&record).Error
}

func (store *Store) ListAuditEvents(ctx context.Context, limit int) ([]AuditEvent, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var total int64
	if err := store.db.WithContext(ctx).Model(&AuditEventRecord{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var records []AuditEventRecord
	if err := store.db.WithContext(ctx).
		Order("created_at DESC").
		Limit(limit).
		Find(&records).Error; err != nil {
		return nil, 0, err
	}
	events := make([]AuditEvent, 0, len(records))
	for _, record := range records {
		events = append(events, AuditEvent{
			ID:         record.ID,
			Action:     record.Action,
			Identifier: record.Identifier,
			Status:     record.Status,
			RequestID:  record.RequestID,
			Source:     record.Source,
			RemoteAddr: record.RemoteAddr,
			CreatedAt:  record.CreatedAt,
		})
	}
	return events, total, nil
}

func (store *Store) ExportCatalog(ctx context.Context, host *ard.HostInfo) (ard.Catalog, error) {
	var records []CatalogEntryRecord
	if err := store.db.WithContext(ctx).
		Where("lifecycle_status = ?", LifecycleStatusActive).
		Order("display_name ASC").
		Find(&records).Error; err != nil {
		return ard.Catalog{}, err
	}
	entries := make([]ard.CatalogEntry, 0, len(records))
	for _, record := range records {
		entry, err := record.ToCatalogEntry()
		if err != nil {
			return ard.Catalog{}, err
		}
		entries = append(entries, entry)
	}
	catalog := ard.Catalog{
		SpecVersion: "1.0",
		Host:        host,
		Entries:     entries,
	}
	if catalog.Host != nil && catalog.Host.DisplayName == "" {
		catalog.Host = nil
	}
	return catalog, nil
}

func (store *Store) Explore(ctx context.Context, request ard.ExploreRequest) (ard.ExploreResponse, error) {
	records, err := store.matchingRecords(ctx, request.Query, "", 0)
	if err != nil {
		return ard.ExploreResponse{}, err
	}
	entries := make([]ard.CatalogEntry, 0, len(records))
	for _, record := range records {
		entry, err := record.ToCatalogEntry()
		if err != nil {
			return ard.ExploreResponse{}, err
		}
		if matchesFilter(entry, request.Query.Filter) {
			entries = append(entries, entry)
		}
	}

	facets := make(map[string]ard.ExploreFacet, len(request.ResultType.Facets))
	for _, facetRequest := range request.ResultType.Facets {
		if facetRequest.Field == "" {
			continue
		}
		facets[facetRequest.Field] = buildFacet(entries, facetRequest)
	}
	return ard.ExploreResponse{ResultType: "facets", Facets: facets}, nil
}

func (store *Store) Count(ctx context.Context) (int64, error) {
	var count int64
	err := store.db.WithContext(ctx).Model(&CatalogEntryRecord{}).
		Where("lifecycle_status = ?", LifecycleStatusActive).
		Count(&count).Error
	return count, err
}

func (store *Store) matchingRecords(ctx context.Context, searchQuery ard.SearchQuery, source string, limit int) ([]CatalogEntryRecord, error) {
	query := store.db.WithContext(ctx).Model(&CatalogEntryRecord{}).
		Where("lifecycle_status = ?", LifecycleStatusActive).
		Order("display_name ASC")
	if source != "" {
		query = query.Where("source = ?", source)
	}
	if searchQuery.Text != "" {
		terms := strings.Fields(strings.ToLower(searchQuery.Text))
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
	if types := searchQuery.Filter["type"]; len(types) > 0 {
		query = query.Where("type IN ?", types)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	var records []CatalogEntryRecord
	if err := query.Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

func recordFromEntry(entry ard.CatalogEntry, source string) (CatalogEntryRecord, error) {
	return recordFromEntryWithStatus(entry, source, LifecycleStatusActive)
}

func recordFromEntryWithStatus(entry ard.CatalogEntry, source string, status string) (CatalogEntryRecord, error) {
	normalizedStatus, err := NormalizeLifecycleStatus(status)
	if err != nil {
		return CatalogEntryRecord{}, err
	}
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
		LifecycleStatus:       normalizedStatus,
		SearchText:            searchText(entry),
	}, nil
}

func (record CatalogEntryRecord) ToCatalogEntry() (ard.CatalogEntry, error) {
	return record.toCatalogEntry(false)
}

func (record CatalogEntryRecord) toCatalogEntry(includeLifecycleMetadata bool) (ard.CatalogEntry, error) {
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
	if includeLifecycleMetadata {
		if entry.Metadata == nil {
			entry.Metadata = map[string]any{}
		}
		entry.Metadata["ard.status"] = record.NormalizedLifecycleStatus()
	}
	return entry, nil
}

func (record CatalogEntryRecord) NormalizedLifecycleStatus() string {
	status, err := NormalizeLifecycleStatus(record.LifecycleStatus)
	if err != nil {
		return LifecycleStatusActive
	}
	return status
}

func NormalizeLifecycleStatus(status string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(status))
	switch normalized {
	case LifecycleStatusActive, LifecycleStatusPending, LifecycleStatusDisabled:
		return normalized, nil
	default:
		return "", fmt.Errorf("status must be one of: %s, %s, %s", LifecycleStatusActive, LifecycleStatusPending, LifecycleStatusDisabled)
	}
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

func buildFacet(entries []ard.CatalogEntry, request ard.ExploreFacetRequest) ard.ExploreFacet {
	limit := request.Limit
	if limit <= 0 {
		limit = 20
	}
	minCount := request.MinCount
	if minCount <= 0 {
		minCount = 1
	}

	counts := map[string]int{}
	for _, entry := range entries {
		for _, value := range facetValues(entry, request.Field) {
			counts[value]++
		}
	}

	buckets := make([]ard.ExploreFacetBucket, 0, len(counts))
	for value, count := range counts {
		if count >= minCount {
			buckets = append(buckets, ard.ExploreFacetBucket{Value: value, Count: count})
		}
	}
	sort.Slice(buckets, func(left int, right int) bool {
		if buckets[left].Count == buckets[right].Count {
			return buckets[left].Value < buckets[right].Value
		}
		return buckets[left].Count > buckets[right].Count
	})

	otherCount := 0
	if len(buckets) > limit {
		for _, bucket := range buckets[limit:] {
			otherCount += bucket.Count
		}
		buckets = buckets[:limit]
	}
	return ard.ExploreFacet{Buckets: buckets, OtherCount: otherCount}
}

func facetValues(entry ard.CatalogEntry, field string) []string {
	switch field {
	case "type":
		return []string{entry.Type}
	case "publisher":
		return []string{ard.Publisher(entry.Identifier)}
	case "tags":
		return entry.Tags
	case "capabilities":
		return entry.Capabilities
	default:
		if strings.HasPrefix(field, "metadata.") {
			key := strings.TrimPrefix(field, "metadata.")
			if value, ok := entry.Metadata[key].(string); ok {
				return []string{value}
			}
		}
		return nil
	}
}

func FormatCatalogImport(count int, source string) string {
	return fmt.Sprintf("imported %d catalog entries from %s", count, source)
}
