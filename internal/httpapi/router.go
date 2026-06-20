package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ifuryst/ard/internal/ard"
	"github.com/ifuryst/ard/internal/federation"
	"github.com/ifuryst/ard/internal/pagination"
	"github.com/ifuryst/ard/internal/policy"
	"github.com/ifuryst/ard/internal/store"
)

type Server struct {
	store            *store.Store
	adminAuthorizer  *adminAuthorizer
	policy           *policy.Policy
	metricsCollector *metricsCollector
}

type Options struct {
	AdminToken      string
	AdminTokens     []AdminToken
	AdminTokensFile string
	Policy          *policy.Policy
}

func NewRouter(store *store.Store) *gin.Engine {
	return NewRouterWithOptions(store, Options{})
}

func NewRouterWithOptions(store *store.Store, options Options) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	adminTokens := options.AdminTokens
	if token := strings.TrimSpace(options.AdminToken); token != "" {
		adminTokens = append(adminTokens, AdminToken{
			Name:  "default-admin",
			Token: token,
			Role:  adminRoleAdmin,
		})
	}
	server := Server{
		store:            store,
		adminAuthorizer:  newAdminAuthorizer(adminTokens, options.AdminTokensFile),
		policy:           options.Policy,
		metricsCollector: newMetricsCollector(),
	}
	router := gin.New()
	router.Use(requestIDMiddleware(), traceContextMiddleware(), metricsMiddleware(server.metricsCollector), jsonAccessLogMiddleware(), gin.Recovery())

	router.GET("/health", server.health)
	router.GET("/metrics", server.metrics)
	router.GET("/.well-known/ai-catalog.json", server.catalog)
	router.GET("/agents", server.agents)
	router.POST("/search", server.search)
	router.POST("/explore", server.explore)
	if server.adminAuthorizer != nil {
		admin := router.Group("/admin")
		admin.GET("/audit/verify", server.requireAdminPermission(adminPermissionRead), server.adminVerifyAuditChain)
		admin.GET("/audit", server.requireAdminPermission(adminPermissionRead), server.adminAuditEvents)
		admin.GET("/reviews", server.requireAdminPermission(adminPermissionRead), server.adminReviewEntries)
		admin.GET("/entries", server.requireAdminPermission(adminPermissionRead), server.adminEntries)
		admin.GET("/catalog", server.requireAdminPermission(adminPermissionRead), server.adminExportCatalog)
		admin.POST("/entries", server.requireAdminPermission(adminPermissionPublish), server.adminUpsertEntry)
		admin.POST("/catalogs", server.requireAdminPermission(adminPermissionPublish), server.adminUpsertCatalog)
		admin.POST("/reviews/:identifier/approve", server.requireAdminPermission(adminPermissionReview), server.adminApproveReview)
		admin.POST("/reviews/:identifier/reject", server.requireAdminPermission(adminPermissionReview), server.adminRejectReview)
		admin.PATCH("/entries/:identifier/status", server.requireAdminPermission(adminPermissionOperate), server.adminSetEntryStatus)
		admin.DELETE("/entries/:identifier", server.requireAdminPermission(adminPermissionOperate), server.adminDeleteEntry)
	}
	return router
}

func (server Server) health(context *gin.Context) {
	count, err := server.store.Count(context.Request.Context())
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}
	context.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"entries": count,
	})
}

func (server Server) catalog(context *gin.Context) {
	baseURL := requestBaseURL(context.Request)
	catalog, err := server.store.ExportCatalog(context.Request.Context(), registryHostInfo())
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{
			"errorCode": "INTERNAL_ERROR",
			"message":   err.Error(),
		})
		return
	}
	catalog.Entries = prependRegistryEntry(catalog.Entries, registryCatalogEntry(baseURL))
	context.JSON(http.StatusOK, catalog)
}

func registryHostInfo() *ard.HostInfo {
	return &ard.HostInfo{
		DisplayName:      "ARD",
		Identifier:       "did:web:agent.localhost",
		DocumentationURL: "https://github.com/iFurySt/ard",
	}
}

func registryCatalogEntry(baseURL string) ard.CatalogEntry {
	return ard.CatalogEntry{
		Identifier:  "urn:air:agent.localhost:registry:ard",
		DisplayName: "ARD Registry",
		Type:        ard.TypeAIRegistry,
		URL:         baseURL,
		Description: "Self-hosted Agentic Resource Discovery registry.",
		Tags:        []string{"ard", "registry", "self-hosted"},
	}
}

func prependRegistryEntry(entries []ard.CatalogEntry, registry ard.CatalogEntry) []ard.CatalogEntry {
	for _, entry := range entries {
		if entry.Identifier == registry.Identifier {
			return entries
		}
	}
	published := make([]ard.CatalogEntry, 0, len(entries)+1)
	published = append(published, registry)
	published = append(published, entries...)
	return published
}

func (server Server) search(context *gin.Context) {
	var request ard.SearchRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{
			"errorCode": "INVALID_ARGUMENT",
			"message":   err.Error(),
		})
		return
	}
	if err := ard.ValidateSearchRequest(request); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{
			"errorCode": "INVALID_ARGUMENT",
			"message":   err.Error(),
		})
		return
	}
	if request.NormalizedFederation() == "auto" {
		server.autoFederationSearch(context, request)
		return
	}
	page, err := server.store.SearchPage(context.Request.Context(), request, "")
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidToken) {
			context.JSON(http.StatusBadRequest, gin.H{
				"errorCode": "INVALID_ARGUMENT",
				"message":   err.Error(),
			})
			return
		}
		context.JSON(http.StatusInternalServerError, gin.H{
			"errorCode": "INTERNAL_ERROR",
			"message":   err.Error(),
		})
		return
	}
	response := ard.SearchResponse{Results: page.Results, PageToken: page.NextPageToken}
	switch request.NormalizedFederation() {
	case "referrals":
		referrals, err := server.store.RegistryReferrals(context.Request.Context(), request.NormalizedPageSize())
		if err != nil {
			context.JSON(http.StatusInternalServerError, gin.H{
				"errorCode": "INTERNAL_ERROR",
				"message":   err.Error(),
			})
			return
		}
		response.Referrals = referrals
	}
	context.JSON(http.StatusOK, response)
}

func (server Server) autoFederationSearch(context *gin.Context, request ard.SearchRequest) {
	state, err := decodeAutoFederationPageToken(request.PageToken)
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidToken) {
			context.JSON(http.StatusBadRequest, gin.H{
				"errorCode": "INVALID_ARGUMENT",
				"message":   err.Error(),
			})
			return
		}
		context.JSON(http.StatusInternalServerError, gin.H{
			"errorCode": "INTERNAL_ERROR",
			"message":   err.Error(),
		})
		return
	}
	localPage := store.SearchPage{}
	if state.Initial || state.LocalPageToken != "" {
		localRequest := request
		localRequest.PageToken = state.LocalPageToken
		localPage, err = server.store.SearchPage(context.Request.Context(), localRequest, "")
		if err != nil {
			if errors.Is(err, pagination.ErrInvalidToken) {
				context.JSON(http.StatusBadRequest, gin.H{
					"errorCode": "INVALID_ARGUMENT",
					"message":   err.Error(),
				})
				return
			}
			context.JSON(http.StatusInternalServerError, gin.H{
				"errorCode": "INTERNAL_ERROR",
				"message":   err.Error(),
			})
			return
		}
	}
	referrals, err := server.store.RegistryReferrals(context.Request.Context(), federation.MaxUpstreamRegistries)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{
			"errorCode": "INTERNAL_ERROR",
			"message":   err.Error(),
		})
		return
	}
	var upstreamTokens map[string]string
	if !state.Initial {
		upstreamTokens = state.UpstreamPageToken
	}
	upstreamPage := federation.NewClient().SearchPage(context.Request.Context(), referrals, request, upstreamTokens)
	results, buffered := mergeSearchResultsPage(state.Buffered, localPage.Results, upstreamPage.Results, request.NormalizedPageSize())
	nextState := autoFederationPageState{
		LocalPageToken:    localPage.NextPageToken,
		UpstreamPageToken: upstreamPage.NextPageTokens,
		Buffered:          buffered,
	}
	context.JSON(http.StatusOK, ard.SearchResponse{
		Results:   results,
		PageToken: encodeAutoFederationPageToken(nextState),
	})
}

func mergeSearchResults(local []ard.SearchResult, upstream []ard.SearchResult, limit int) []ard.SearchResult {
	results, _ := mergeSearchResultsPage(nil, local, upstream, limit)
	return results
}

func mergeSearchResultsPage(buffered []autoFederationBufferedResult, local []ard.SearchResult, upstream []ard.SearchResult, limit int) ([]ard.SearchResult, []autoFederationBufferedResult) {
	if limit <= 0 {
		limit = 10
	}
	type candidate struct {
		result ard.SearchResult
		local  bool
		order  int
	}
	seen := map[string]int{}
	candidates := make([]candidate, 0, len(local)+len(upstream))
	appendResult := func(result ard.SearchResult, local bool) {
		candidate := candidate{
			result: result,
			local:  local,
			order:  len(candidates),
		}
		if result.Identifier != "" {
			existingIndex, ok := seen[result.Identifier]
			if ok {
				if local && !candidates[existingIndex].local {
					candidate.order = candidates[existingIndex].order
					candidates[existingIndex] = candidate
				}
				return
			}
			seen[result.Identifier] = len(candidates)
		}
		candidates = append(candidates, candidate)
	}
	for _, bufferedResult := range buffered {
		appendResult(bufferedResult.Result, bufferedResult.Local)
	}
	for _, result := range local {
		appendResult(result, true)
	}
	for _, result := range upstream {
		appendResult(result, false)
	}
	sort.SliceStable(candidates, func(i int, j int) bool {
		left := candidates[i]
		right := candidates[j]
		if left.result.Score != right.result.Score {
			return left.result.Score > right.result.Score
		}
		if left.local != right.local {
			return left.local
		}
		if left.result.Identifier != right.result.Identifier {
			return left.result.Identifier < right.result.Identifier
		}
		if left.result.DisplayName != right.result.DisplayName {
			return left.result.DisplayName < right.result.DisplayName
		}
		return left.order < right.order
	})
	returned := candidates
	remaining := []candidate{}
	if len(candidates) > limit {
		returned = candidates[:limit]
		remaining = candidates[limit:]
	}
	results := make([]ard.SearchResult, 0, len(returned))
	for _, candidate := range returned {
		results = append(results, candidate.result)
	}
	nextBuffered := make([]autoFederationBufferedResult, 0, len(remaining))
	for _, candidate := range remaining {
		nextBuffered = append(nextBuffered, autoFederationBufferedResult{
			Result: candidate.result,
			Local:  candidate.local,
		})
	}
	return results, nextBuffered
}

func (server Server) agents(context *gin.Context) {
	options, err := parseAgentsListOptions(context)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{
			"errorCode": "INVALID_ARGUMENT",
			"message":   err.Error(),
		})
		return
	}
	page, err := server.store.ListEntriesPage(context.Request.Context(), options)
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidToken) {
			context.JSON(http.StatusBadRequest, gin.H{
				"errorCode": "INVALID_ARGUMENT",
				"message":   err.Error(),
			})
			return
		}
		context.JSON(http.StatusInternalServerError, gin.H{
			"errorCode": "INTERNAL_ERROR",
			"message":   err.Error(),
		})
		return
	}
	context.JSON(http.StatusOK, ard.ListResponse{
		Items:     page.Entries,
		Total:     int(page.Total),
		PageToken: page.NextPageToken,
	})
}

func parseAgentsListOptions(context *gin.Context) (store.ListOptions, error) {
	for parameter := range context.Request.URL.Query() {
		switch parameter {
		case "pageSize", "pageToken", "filter", "orderBy":
			continue
		default:
			return store.ListOptions{}, fmt.Errorf("unsupported query parameter %q", parameter)
		}
	}

	limit := 20
	if rawPageSize := strings.TrimSpace(context.Query("pageSize")); rawPageSize != "" {
		parsed, err := strconv.Atoi(rawPageSize)
		if err != nil {
			return store.ListOptions{}, errors.New("pageSize must be an integer")
		}
		if parsed < 1 || parsed > 100 {
			return store.ListOptions{}, errors.New("pageSize must be between 1 and 100")
		}
		limit = parsed
	}
	filter, err := store.ParseListFilterExpression(context.Query("filter"))
	if err != nil {
		return store.ListOptions{}, err
	}
	orderBy, err := store.ParseListOrderBy(context.Query("orderBy"))
	if err != nil {
		return store.ListOptions{}, err
	}
	return store.ListOptions{
		Limit:     limit,
		PageToken: context.Query("pageToken"),
		Filter:    filter,
		OrderBy:   orderBy,
	}, nil
}

func (server Server) explore(context *gin.Context) {
	var request ard.ExploreRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{
			"errorCode": "INVALID_ARGUMENT",
			"message":   err.Error(),
		})
		return
	}
	if err := ard.ValidateExploreRequest(request); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{
			"errorCode": "INVALID_ARGUMENT",
			"message":   err.Error(),
		})
		return
	}
	response, err := server.store.Explore(context.Request.Context(), request)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{
			"errorCode": "INTERNAL_ERROR",
			"message":   err.Error(),
		})
		return
	}
	context.JSON(http.StatusOK, response)
}

func (server Server) adminEntries(context *gin.Context) {
	limit, _ := strconv.Atoi(context.DefaultQuery("pageSize", "20"))
	mediaType := context.Query("type")
	if mediaType == "" {
		mediaType = mediaTypeForKind(context.Query("kind"))
	}
	status := strings.TrimSpace(context.Query("status"))
	if status != "" {
		normalized, err := store.NormalizeLifecycleStatus(status)
		if err != nil {
			context.JSON(http.StatusBadRequest, gin.H{
				"errorCode": "INVALID_ARGUMENT",
				"message":   err.Error(),
			})
			return
		}
		status = normalized
	}
	page, err := server.store.ListEntriesPage(context.Request.Context(), store.ListOptions{
		Limit:                    limit,
		PageToken:                context.Query("pageToken"),
		Type:                     mediaType,
		Status:                   status,
		IncludeInactive:          status == "",
		IncludeLifecycleMetadata: true,
	})
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidToken) {
			context.JSON(http.StatusBadRequest, gin.H{
				"errorCode": "INVALID_ARGUMENT",
				"message":   err.Error(),
			})
			return
		}
		context.JSON(http.StatusInternalServerError, gin.H{
			"errorCode": "INTERNAL_ERROR",
			"message":   err.Error(),
		})
		return
	}
	context.JSON(http.StatusOK, ard.ListResponse{Items: page.Entries, Total: int(page.Total), PageToken: page.NextPageToken})
}

func (server Server) adminReviewEntries(context *gin.Context) {
	limit, _ := strconv.Atoi(context.DefaultQuery("pageSize", "20"))
	page, err := server.store.ListEntriesPage(context.Request.Context(), store.ListOptions{
		Limit:                    limit,
		PageToken:                context.Query("pageToken"),
		Status:                   store.LifecycleStatusPending,
		IncludeLifecycleMetadata: true,
	})
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidToken) {
			context.JSON(http.StatusBadRequest, gin.H{
				"errorCode": "INVALID_ARGUMENT",
				"message":   err.Error(),
			})
			return
		}
		context.JSON(http.StatusInternalServerError, gin.H{
			"errorCode": "INTERNAL_ERROR",
			"message":   err.Error(),
		})
		return
	}
	context.JSON(http.StatusOK, ard.ListResponse{Items: page.Entries, Total: int(page.Total), PageToken: page.NextPageToken})
}

func (server Server) adminUpsertEntry(context *gin.Context) {
	var entry ard.CatalogEntry
	if err := context.ShouldBindJSON(&entry); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{
			"errorCode": "INVALID_ARGUMENT",
			"message":   err.Error(),
		})
		return
	}
	catalog := ard.Catalog{SpecVersion: "1.0", Entries: []ard.CatalogEntry{entry}}
	if err := ard.ValidateCatalog(catalog); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{
			"errorCode": "INVALID_ARGUMENT",
			"message":   err.Error(),
		})
		return
	}
	statuses, err := server.evaluatePolicy(catalog)
	if err != nil {
		server.writePolicyError(context, err)
		return
	}
	if err := server.store.UpsertCatalogWithStatuses(context.Request.Context(), catalog, "admin-api", statuses); err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{
			"errorCode": "INTERNAL_ERROR",
			"message":   err.Error(),
		})
		return
	}
	if err := server.recordAuditEvent(context, "entry.upsert", entry.Identifier, statuses[entry.Identifier]); err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{
			"errorCode": "INTERNAL_ERROR",
			"message":   err.Error(),
		})
		return
	}
	context.JSON(http.StatusCreated, entry)
}

func (server Server) adminUpsertCatalog(context *gin.Context) {
	var catalog ard.Catalog
	if err := context.ShouldBindJSON(&catalog); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{
			"errorCode": "INVALID_ARGUMENT",
			"message":   err.Error(),
		})
		return
	}
	if err := ard.ValidateCatalog(catalog); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{
			"errorCode": "INVALID_ARGUMENT",
			"message":   err.Error(),
		})
		return
	}
	statuses, err := server.evaluatePolicy(catalog)
	if err != nil {
		server.writePolicyError(context, err)
		return
	}
	if err := server.store.UpsertCatalogWithStatuses(context.Request.Context(), catalog, "admin-api", statuses); err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{
			"errorCode": "INTERNAL_ERROR",
			"message":   err.Error(),
		})
		return
	}
	for _, entry := range catalog.Entries {
		if err := server.recordAuditEvent(context, "catalog.upsert", entry.Identifier, statuses[entry.Identifier]); err != nil {
			context.JSON(http.StatusInternalServerError, gin.H{
				"errorCode": "INTERNAL_ERROR",
				"message":   err.Error(),
			})
			return
		}
	}
	context.JSON(http.StatusCreated, gin.H{
		"entries": len(catalog.Entries),
	})
}

func (server Server) adminAuditEvents(context *gin.Context) {
	limit, _ := strconv.Atoi(context.DefaultQuery("pageSize", "50"))
	page, err := server.store.ListAuditEventsPage(context.Request.Context(), limit, context.Query("pageToken"))
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidToken) {
			context.JSON(http.StatusBadRequest, gin.H{
				"errorCode": "INVALID_ARGUMENT",
				"message":   err.Error(),
			})
			return
		}
		context.JSON(http.StatusInternalServerError, gin.H{
			"errorCode": "INTERNAL_ERROR",
			"message":   err.Error(),
		})
		return
	}
	context.JSON(http.StatusOK, gin.H{
		"items":     page.Events,
		"total":     page.Total,
		"pageToken": page.NextPageToken,
	})
}

func (server Server) adminVerifyAuditChain(context *gin.Context) {
	result, err := server.store.VerifyAuditChain(context.Request.Context())
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{
			"errorCode": "INTERNAL_ERROR",
			"message":   err.Error(),
		})
		return
	}
	context.JSON(http.StatusOK, result)
}

func (server Server) evaluatePolicy(catalog ard.Catalog) (map[string]string, error) {
	if server.policy == nil {
		return nil, nil
	}
	statuses, _, err := server.policy.EvaluateCatalog(catalog)
	return statuses, err
}

func (server Server) writePolicyError(context *gin.Context, err error) {
	var denied policy.DeniedError
	if errors.As(err, &denied) {
		context.JSON(http.StatusForbidden, gin.H{
			"errorCode":  "POLICY_DENIED",
			"message":    denied.Error(),
			"identifier": denied.Identifier,
		})
		return
	}
	context.JSON(http.StatusBadRequest, gin.H{
		"errorCode": "POLICY_INVALID",
		"message":   err.Error(),
	})
}

func (server Server) adminExportCatalog(context *gin.Context) {
	catalog, err := server.store.ExportCatalog(context.Request.Context(), &ard.HostInfo{
		DisplayName: "ARD Registry",
	})
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{
			"errorCode": "INTERNAL_ERROR",
			"message":   err.Error(),
		})
		return
	}
	context.JSON(http.StatusOK, catalog)
}

func (server Server) adminSetEntryStatus(context *gin.Context) {
	identifier := context.Param("identifier")
	if err := ard.ValidateIdentifier(identifier); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{
			"errorCode": "INVALID_ARGUMENT",
			"message":   err.Error(),
		})
		return
	}
	var payload struct {
		Status string `json:"status"`
	}
	if err := context.ShouldBindJSON(&payload); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{
			"errorCode": "INVALID_ARGUMENT",
			"message":   err.Error(),
		})
		return
	}
	status, err := store.NormalizeLifecycleStatus(payload.Status)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{
			"errorCode": "INVALID_ARGUMENT",
			"message":   err.Error(),
		})
		return
	}
	updated, err := server.store.SetEntryStatus(context.Request.Context(), identifier, status)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{
			"errorCode": "INTERNAL_ERROR",
			"message":   err.Error(),
		})
		return
	}
	if !updated {
		context.JSON(http.StatusNotFound, gin.H{
			"errorCode": "NOT_FOUND",
			"message":   "entry not found",
		})
		return
	}
	if err := server.recordAuditEvent(context, "entry.status", identifier, status); err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{
			"errorCode": "INTERNAL_ERROR",
			"message":   err.Error(),
		})
		return
	}
	context.JSON(http.StatusOK, gin.H{
		"identifier": identifier,
		"status":     status,
	})
}

func (server Server) adminApproveReview(context *gin.Context) {
	server.adminReviewApprove(context)
}

func (server Server) adminRejectReview(context *gin.Context) {
	server.adminReviewDecision(context, store.LifecycleStatusDisabled, "entry.review.reject")
}

func (server Server) adminReviewDecision(context *gin.Context, status string, action string) {
	identifier := context.Param("identifier")
	reason, ok := server.reviewDecisionPayload(context)
	if !ok {
		return
	}
	if !server.ensurePendingReviewEntry(context, identifier) {
		return
	}
	updated, err := server.store.SetEntryStatus(context.Request.Context(), identifier, status)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{
			"errorCode": "INTERNAL_ERROR",
			"message":   err.Error(),
		})
		return
	}
	if !updated {
		context.JSON(http.StatusNotFound, gin.H{
			"errorCode": "NOT_FOUND",
			"message":   "entry not found",
		})
		return
	}
	if err := server.recordAuditEventWithReason(context, action, identifier, status, reason); err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{
			"errorCode": "INTERNAL_ERROR",
			"message":   err.Error(),
		})
		return
	}
	context.JSON(http.StatusOK, gin.H{
		"identifier": identifier,
		"reason":     reason,
		"status":     status,
	})
}

func (server Server) adminReviewApprove(context *gin.Context) {
	identifier := context.Param("identifier")
	reason, ok := server.reviewDecisionPayload(context)
	if !ok {
		return
	}
	if !server.ensurePendingReviewEntry(context, identifier) {
		return
	}
	requiredApprovals := server.requiredReviewApprovals()
	approval, err := server.store.RecordReviewApproval(
		context.Request.Context(),
		identifier,
		adminReviewerFromContext(context),
		reason,
		requestIDFromContext(context),
	)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{
			"errorCode": "INTERNAL_ERROR",
			"message":   err.Error(),
		})
		return
	}
	if approval.Duplicate {
		context.JSON(http.StatusConflict, gin.H{
			"errorCode":         "FAILED_PRECONDITION",
			"message":           "reviewer already approved this entry",
			"identifier":        identifier,
			"approvals":         approval.Approvals,
			"requiredApprovals": requiredApprovals,
			"status":            store.LifecycleStatusPending,
		})
		return
	}
	status := store.LifecycleStatusPending
	if approval.Approvals >= int64(requiredApprovals) {
		status = store.LifecycleStatusActive
		updated, err := server.store.SetEntryStatus(context.Request.Context(), identifier, status)
		if err != nil {
			context.JSON(http.StatusInternalServerError, gin.H{
				"errorCode": "INTERNAL_ERROR",
				"message":   err.Error(),
			})
			return
		}
		if !updated {
			context.JSON(http.StatusNotFound, gin.H{
				"errorCode": "NOT_FOUND",
				"message":   "entry not found",
			})
			return
		}
	}
	if err := server.recordAuditEventWithReason(context, "entry.review.approve", identifier, status, reason); err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{
			"errorCode": "INTERNAL_ERROR",
			"message":   err.Error(),
		})
		return
	}
	context.JSON(http.StatusOK, gin.H{
		"identifier":        identifier,
		"reason":            reason,
		"status":            status,
		"approvals":         approval.Approvals,
		"requiredApprovals": requiredApprovals,
	})
}

func (server Server) reviewDecisionPayload(context *gin.Context) (string, bool) {
	payload := struct {
		Reason string `json:"reason,omitempty"`
	}{}
	if context.Request.Body != nil && context.Request.ContentLength != 0 {
		if err := context.ShouldBindJSON(&payload); err != nil {
			context.JSON(http.StatusBadRequest, gin.H{
				"errorCode": "INVALID_ARGUMENT",
				"message":   err.Error(),
			})
			return "", false
		}
	}
	reason, ok := normalizeReviewReason(payload.Reason)
	if !ok {
		context.JSON(http.StatusBadRequest, gin.H{
			"errorCode": "INVALID_ARGUMENT",
			"message":   "reason must be 1000 characters or fewer",
		})
		return "", false
	}
	return reason, true
}

func (server Server) ensurePendingReviewEntry(context *gin.Context, identifier string) bool {
	if err := ard.ValidateIdentifier(identifier); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{
			"errorCode": "INVALID_ARGUMENT",
			"message":   err.Error(),
		})
		return false
	}
	entry, found, err := server.store.GetEntry(context.Request.Context(), identifier, true)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{
			"errorCode": "INTERNAL_ERROR",
			"message":   err.Error(),
		})
		return false
	}
	if !found {
		context.JSON(http.StatusNotFound, gin.H{
			"errorCode": "NOT_FOUND",
			"message":   "entry not found",
		})
		return false
	}
	if entry.Metadata["ard.status"] != store.LifecycleStatusPending {
		context.JSON(http.StatusConflict, gin.H{
			"errorCode": "FAILED_PRECONDITION",
			"message":   "entry is not pending review",
		})
		return false
	}
	return true
}

func (server Server) requiredReviewApprovals() int {
	if server.policy == nil {
		return 1
	}
	return server.policy.NormalizedRequiredApprovals()
}

func adminReviewerFromContext(context *gin.Context) string {
	value, ok := context.Get(adminPrincipalKey)
	if !ok {
		return "unknown"
	}
	principal, ok := value.(adminPrincipal)
	if !ok || strings.TrimSpace(principal.Name) == "" {
		return "unknown"
	}
	return principal.Name
}

func normalizeReviewReason(reason string) (string, bool) {
	reason = strings.TrimSpace(reason)
	if len(reason) > 1000 {
		return "", false
	}
	return reason, true
}

func (server Server) adminDeleteEntry(context *gin.Context) {
	identifier := context.Param("identifier")
	if err := ard.ValidateIdentifier(identifier); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{
			"errorCode": "INVALID_ARGUMENT",
			"message":   err.Error(),
		})
		return
	}
	removed, err := server.store.DeleteEntry(context.Request.Context(), identifier)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{
			"errorCode": "INTERNAL_ERROR",
			"message":   err.Error(),
		})
		return
	}
	if !removed {
		context.JSON(http.StatusNotFound, gin.H{
			"errorCode": "NOT_FOUND",
			"message":   "entry not found",
		})
		return
	}
	if err := server.recordAuditEvent(context, "entry.delete", identifier, ""); err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{
			"errorCode": "INTERNAL_ERROR",
			"message":   err.Error(),
		})
		return
	}
	context.Status(http.StatusNoContent)
}

func (server Server) recordAuditEvent(context *gin.Context, action string, identifier string, status string) error {
	return server.recordAuditEventWithReason(context, action, identifier, status, "")
}

func (server Server) recordAuditEventWithReason(context *gin.Context, action string, identifier string, status string, reason string) error {
	return server.store.RecordAuditEvent(context.Request.Context(), store.AuditEvent{
		Action:     action,
		Identifier: identifier,
		Status:     status,
		Reason:     reason,
		RequestID:  requestIDFromContext(context),
		Source:     "admin-api",
		RemoteAddr: context.ClientIP(),
	})
}

func (server Server) requireAdminPermission(permission adminPermission) gin.HandlerFunc {
	return server.adminAuthorizer.require(permission)
}

func mediaTypeForKind(kind string) string {
	switch kind {
	case "mcp":
		return ard.TypeMCPServerCard
	case "a2a":
		return ard.TypeA2AAgentCard
	case "skill":
		return ard.TypeAISkill
	case "catalog":
		return ard.TypeAICatalog
	case "registry":
		return ard.TypeAIRegistry
	case "openapi":
		return ard.TypeOpenAPI
	default:
		return kind
	}
}

func requestBaseURL(request *http.Request) string {
	scheme := "http"
	if request.TLS != nil {
		scheme = "https"
	}
	if forwarded := request.Header.Get("X-Forwarded-Proto"); forwarded != "" {
		scheme = forwarded
	}
	host := request.Host
	if forwardedHost := request.Header.Get("X-Forwarded-Host"); forwardedHost != "" {
		host = forwardedHost
	}
	return scheme + "://" + host
}
