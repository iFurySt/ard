// Package ard exposes the public Agentic Resource Discovery data model used by
// the registry, CLI, and Go client SDK.
package ard

import internalard "github.com/ifuryst/ard/internal/ard"

const (
	TypeAICatalog      = internalard.TypeAICatalog
	TypeAIRegistry     = internalard.TypeAIRegistry
	TypeAIRegistryBare = internalard.TypeAIRegistryBare
	TypeA2AAgentCard   = internalard.TypeA2AAgentCard
	TypeMCPServerCard  = internalard.TypeMCPServerCard
	TypeAISkill        = internalard.TypeAISkill
	TypeOpenAPI        = internalard.TypeOpenAPI
)

type Catalog = internalard.Catalog
type HostInfo = internalard.HostInfo
type CatalogEntry = internalard.CatalogEntry
type Filter = internalard.Filter
type SearchQuery = internalard.SearchQuery
type SearchRequest = internalard.SearchRequest
type SearchResult = internalard.SearchResult
type SearchResponse = internalard.SearchResponse
type ExploreRequest = internalard.ExploreRequest
type ExploreResultType = internalard.ExploreResultType
type ExploreFacetRequest = internalard.ExploreFacetRequest
type ExploreResponse = internalard.ExploreResponse
type ExploreFacet = internalard.ExploreFacet
type ExploreFacetBucket = internalard.ExploreFacetBucket
type ListResponse = internalard.ListResponse

func ValidateCatalog(catalog Catalog) error {
	return internalard.ValidateCatalog(catalog)
}

func ValidateCatalogEntry(entry CatalogEntry) error {
	return internalard.ValidateCatalogEntry(entry)
}

func ValidateSearchRequest(request SearchRequest) error {
	return internalard.ValidateSearchRequest(request)
}

func ValidateExploreRequest(request ExploreRequest) error {
	return internalard.ValidateExploreRequest(request)
}

func ValidateIdentifier(identifier string) error {
	return internalard.ValidateIdentifier(identifier)
}

func Publisher(identifier string) string {
	return internalard.Publisher(identifier)
}
