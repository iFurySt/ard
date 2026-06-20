package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetricsEndpointRecordsRequests(t *testing.T) {
	router := NewRouterWithOptions(nil, Options{})

	missingRequest := httptest.NewRequest(http.MethodGet, "/missing", nil)
	missingResponse := httptest.NewRecorder()
	router.ServeHTTP(missingResponse, missingRequest)
	if missingResponse.Code != http.StatusNotFound {
		t.Fatalf("expected missing route HTTP 404, got %d", missingResponse.Code)
	}

	metricsRequest := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsResponse := httptest.NewRecorder()
	router.ServeHTTP(metricsResponse, metricsRequest)
	if metricsResponse.Code != http.StatusOK {
		t.Fatalf("expected metrics HTTP 200, got %d: %s", metricsResponse.Code, metricsResponse.Body.String())
	}
	contentType := metricsResponse.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/plain") {
		t.Fatalf("expected text/plain metrics response, got %s", contentType)
	}
	body := metricsResponse.Body.String()
	if !strings.Contains(body, "# TYPE ard_http_requests_total counter") {
		t.Fatalf("expected request counter metadata in metrics body: %s", body)
	}
	if !strings.Contains(body, "# TYPE ard_http_request_duration_seconds histogram") {
		t.Fatalf("expected request duration histogram metadata in metrics body: %s", body)
	}
	if !strings.Contains(body, `ard_http_requests_total{method="GET",route="unmatched",status="404"} 1`) {
		t.Fatalf("expected unmatched 404 counter in metrics body: %s", body)
	}
	if !strings.Contains(body, `ard_http_request_duration_seconds_bucket{method="GET",route="unmatched",status="404",le="+Inf"} 1`) {
		t.Fatalf("expected unmatched 404 duration histogram in metrics body: %s", body)
	}
	if !strings.Contains(body, `ard_http_request_duration_seconds_count{method="GET",route="unmatched",status="404"} 1`) {
		t.Fatalf("expected unmatched 404 duration count in metrics body: %s", body)
	}
	if !strings.Contains(body, "ard_http_requests_in_flight") {
		t.Fatalf("expected in-flight gauge in metrics body: %s", body)
	}
	if !strings.Contains(body, "# TYPE ard_runtime_goroutines gauge") {
		t.Fatalf("expected runtime goroutine gauge in metrics body: %s", body)
	}
	if !strings.Contains(body, "# TYPE ard_runtime_heap_alloc_bytes gauge") {
		t.Fatalf("expected runtime heap alloc gauge in metrics body: %s", body)
	}
	if !strings.Contains(body, "# TYPE ard_runtime_gc_cycles_total counter") {
		t.Fatalf("expected runtime GC counter in metrics body: %s", body)
	}
}
