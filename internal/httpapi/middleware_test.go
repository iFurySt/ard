package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestIDMiddlewarePropagatesProvidedID(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(requestIDMiddleware())
	router.GET("/ping", func(context *gin.Context) {
		if got := requestIDFromContext(context); got != "test-request-id" {
			t.Fatalf("unexpected request id in context: %s", got)
		}
		context.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/ping", nil)
	request.Header.Set("X-Request-ID", "test-request-id")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if got := response.Header().Get("X-Request-ID"); got != "test-request-id" {
		t.Fatalf("unexpected response request id: %s", got)
	}
}

func TestRequestIDMiddlewareGeneratesID(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(requestIDMiddleware())
	router.GET("/ping", func(context *gin.Context) {
		if requestIDFromContext(context) == "" {
			t.Fatal("expected generated request id in context")
		}
		context.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/ping", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if got := response.Header().Get("X-Request-ID"); got == "" {
		t.Fatal("expected generated response request id")
	}
}
