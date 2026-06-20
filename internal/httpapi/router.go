package httpapi

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ifuryst/ard/internal/ard"
	"github.com/ifuryst/ard/internal/store"
)

type Server struct {
	store *store.Store
}

func NewRouter(store *store.Store) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	server := Server{store: store}
	router := gin.New()
	router.Use(gin.Recovery())

	router.GET("/health", server.health)
	router.GET("/.well-known/ai-catalog.json", server.catalog)
	router.POST("/search", server.search)
	router.POST("/explore", server.explore)
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

func (server Server) explore(context *gin.Context) {
	context.JSON(http.StatusNotImplemented, gin.H{
		"errorCode": "NOT_IMPLEMENTED",
		"message":   "Explore is not implemented",
	})
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
