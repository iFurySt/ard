package httpapi

import (
	"errors"
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
	context.JSON(http.StatusOK, ard.Catalog{
		SpecVersion: "1.0",
		Host: &ard.HostInfo{
			DisplayName:      "ARD",
			Identifier:       "did:web:agent.localhost",
			DocumentationURL: "https://github.com/iFurySt/ard",
		},
		Entries: []ard.CatalogEntry{
			{
				Identifier:  "urn:air:agent.localhost:registry:ard",
				DisplayName: "ARD Registry",
				Type:        ard.TypeAIRegistry,
				URL:         baseURL,
				Description: "Self-hosted Agentic Resource Discovery registry.",
				Tags:        []string{"ard", "registry", "self-hosted"},
			},
		},
	})
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
	if strings.TrimSpace(request.Query.Text) == "" {
		context.JSON(http.StatusBadRequest, gin.H{
			"errorCode": "INVALID_ARGUMENT",
			"message":   "query.text is required",
		})
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
	case "auto":
		referrals, err := server.store.RegistryReferrals(context.Request.Context(), federation.MaxUpstreamRegistries)
		if err != nil {
			context.JSON(http.StatusInternalServerError, gin.H{
				"errorCode": "INTERNAL_ERROR",
				"message":   err.Error(),
			})
			return
		}
		upstreamResults := federation.NewClient().Search(context.Request.Context(), referrals, request)
		response.Results = mergeSearchResults(page.Results, upstreamResults, request.NormalizedPageSize())
	}
	context.JSON(http.StatusOK, response)
}

func mergeSearchResults(local []ard.SearchResult, upstream []ard.SearchResult, limit int) []ard.SearchResult {
	if limit <= 0 {
		limit = 10
	}
	type candidate struct {
		result ard.SearchResult
		local  bool
		order  int
	}
	seen := map[string]struct{}{}
	candidates := make([]candidate, 0, len(local)+len(upstream))
	appendResult := func(result ard.SearchResult, local bool) {
		if result.Identifier != "" {
			if _, ok := seen[result.Identifier]; ok {
				return
			}
			seen[result.Identifier] = struct{}{}
		}
		candidates = append(candidates, candidate{
			result: result,
			local:  local,
			order:  len(candidates),
		})
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
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	results := make([]ard.SearchResult, 0, len(candidates))
	for _, candidate := range candidates {
		results = append(results, candidate.result)
	}
	return results
}

func (server Server) agents(context *gin.Context) {
	limit, _ := strconv.Atoi(context.DefaultQuery("pageSize", "20"))
	page, err := server.store.ListEntriesPage(context.Request.Context(), store.ListOptions{
		Limit:     limit,
		PageToken: context.Query("pageToken"),
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
	context.JSON(http.StatusOK, ard.ListResponse{
		Items:     page.Entries,
		Total:     int(page.Total),
		PageToken: page.NextPageToken,
	})
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
	if len(request.ResultType.Facets) == 0 {
		context.JSON(http.StatusBadRequest, gin.H{
			"errorCode": "INVALID_ARGUMENT",
			"message":   "resultType.facets is required",
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
	server.adminReviewDecision(context, store.LifecycleStatusActive, "entry.review.approve")
}

func (server Server) adminRejectReview(context *gin.Context) {
	server.adminReviewDecision(context, store.LifecycleStatusDisabled, "entry.review.reject")
}

func (server Server) adminReviewDecision(context *gin.Context, status string, action string) {
	identifier := context.Param("identifier")
	if err := ard.ValidateIdentifier(identifier); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{
			"errorCode": "INVALID_ARGUMENT",
			"message":   err.Error(),
		})
		return
	}
	entry, found, err := server.store.GetEntry(context.Request.Context(), identifier, true)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{
			"errorCode": "INTERNAL_ERROR",
			"message":   err.Error(),
		})
		return
	}
	if !found {
		context.JSON(http.StatusNotFound, gin.H{
			"errorCode": "NOT_FOUND",
			"message":   "entry not found",
		})
		return
	}
	if entry.Metadata["ard.status"] != store.LifecycleStatusPending {
		context.JSON(http.StatusConflict, gin.H{
			"errorCode": "FAILED_PRECONDITION",
			"message":   "entry is not pending review",
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
	if err := server.recordAuditEvent(context, action, identifier, status); err != nil {
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
	return server.store.RecordAuditEvent(context.Request.Context(), store.AuditEvent{
		Action:     action,
		Identifier: identifier,
		Status:     status,
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
