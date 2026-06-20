package httpapi

import (
	"crypto/subtle"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ifuryst/ard/internal/ard"
	"github.com/ifuryst/ard/internal/store"
)

type Server struct {
	store      *store.Store
	adminToken string
}

type Options struct {
	AdminToken string
}

func NewRouter(store *store.Store) *gin.Engine {
	return NewRouterWithOptions(store, Options{})
}

func NewRouterWithOptions(store *store.Store, options Options) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	server := Server{store: store, adminToken: strings.TrimSpace(options.AdminToken)}
	router := gin.New()
	router.Use(requestIDMiddleware(), jsonAccessLogMiddleware(), gin.Recovery())

	router.GET("/health", server.health)
	router.GET("/.well-known/ai-catalog.json", server.catalog)
	router.GET("/agents", server.agents)
	router.POST("/search", server.search)
	router.POST("/explore", server.explore)
	if server.adminToken != "" {
		admin := router.Group("/admin", server.requireAdminToken)
		admin.GET("/audit", server.adminAuditEvents)
		admin.GET("/entries", server.adminEntries)
		admin.POST("/entries", server.adminUpsertEntry)
		admin.POST("/catalogs", server.adminUpsertCatalog)
		admin.GET("/catalog", server.adminExportCatalog)
		admin.PATCH("/entries/:identifier/status", server.adminSetEntryStatus)
		admin.DELETE("/entries/:identifier", server.adminDeleteEntry)
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
	results, err := server.store.Search(context.Request.Context(), request, "")
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{
			"errorCode": "INTERNAL_ERROR",
			"message":   err.Error(),
		})
		return
	}
	context.JSON(http.StatusOK, ard.SearchResponse{Results: results})
}

func (server Server) agents(context *gin.Context) {
	limit, _ := strconv.Atoi(context.DefaultQuery("pageSize", "20"))
	entries, total, err := server.store.List(context.Request.Context(), limit)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{
			"errorCode": "INTERNAL_ERROR",
			"message":   err.Error(),
		})
		return
	}
	context.JSON(http.StatusOK, ard.ListResponse{
		Items: entries,
		Total: int(total),
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
	entries, total, err := server.store.ListEntries(context.Request.Context(), store.ListOptions{
		Limit:                    limit,
		Type:                     mediaType,
		Status:                   status,
		IncludeInactive:          status == "",
		IncludeLifecycleMetadata: true,
	})
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{
			"errorCode": "INTERNAL_ERROR",
			"message":   err.Error(),
		})
		return
	}
	context.JSON(http.StatusOK, ard.ListResponse{Items: entries, Total: int(total)})
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
	if err := server.store.UpsertCatalog(context.Request.Context(), catalog, "admin-api"); err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{
			"errorCode": "INTERNAL_ERROR",
			"message":   err.Error(),
		})
		return
	}
	if err := server.recordAuditEvent(context, "entry.upsert", entry.Identifier, ""); err != nil {
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
	if err := server.store.UpsertCatalog(context.Request.Context(), catalog, "admin-api"); err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{
			"errorCode": "INTERNAL_ERROR",
			"message":   err.Error(),
		})
		return
	}
	for _, entry := range catalog.Entries {
		if err := server.recordAuditEvent(context, "catalog.upsert", entry.Identifier, ""); err != nil {
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
	events, total, err := server.store.ListAuditEvents(context.Request.Context(), limit)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{
			"errorCode": "INTERNAL_ERROR",
			"message":   err.Error(),
		})
		return
	}
	context.JSON(http.StatusOK, gin.H{
		"items": events,
		"total": total,
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

func (server Server) requireAdminToken(context *gin.Context) {
	expected := "Bearer " + server.adminToken
	got := context.GetHeader("Authorization")
	if subtle.ConstantTimeCompare([]byte(got), []byte(expected)) != 1 {
		context.JSON(http.StatusUnauthorized, gin.H{
			"errorCode": "UNAUTHENTICATED",
			"message":   "admin bearer token is required",
		})
		context.Abort()
		return
	}
	context.Next()
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
