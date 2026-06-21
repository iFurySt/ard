package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetricsCommand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/metrics" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		if request.Header.Get("Accept") != "text/plain" {
			t.Fatalf("unexpected accept header: %s", request.Header.Get("Accept"))
		}
		if request.Header.Get("User-Agent") != "ardctl/0.1" {
			t.Fatalf("unexpected user agent: %s", request.Header.Get("User-Agent"))
		}
		response.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = response.Write([]byte("# TYPE ard_http_requests_total counter\nard_http_requests_total 1\n"))
	}))
	t.Cleanup(server.Close)

	command := newMetricsCommand()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs([]string{"--registry-url", server.URL})
	if err := command.Execute(); err != nil {
		t.Fatalf("metrics command: %v", err)
	}
	if got := output.String(); !strings.Contains(got, "ard_http_requests_total") {
		t.Fatalf("unexpected metrics output: %s", got)
	}
}

func TestMetricsCommandReportsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		http.Error(response, "metrics disabled", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	command := newMetricsCommand()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs([]string{"--registry-url", server.URL})
	err := command.Execute()
	if err == nil || !strings.Contains(err.Error(), "metrics disabled") {
		t.Fatalf("expected metrics error, got %v output %s", err, output.String())
	}
}
