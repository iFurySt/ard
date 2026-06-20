package cli

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAdminRequestRequiresToken(t *testing.T) {
	_, err := adminRequest(context.Background(), adminOptions{registryURL: "http://127.0.0.1:1"}, http.MethodGet, "/admin/entries", nil)
	if err == nil {
		t.Fatal("expected missing token error")
	}
	if !strings.Contains(err.Error(), "admin token is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAdminRequestSendsBearerToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if got := request.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("unexpected authorization header: %s", got)
		}
		if got := request.Header.Get("User-Agent"); got != "ardctl/0.1" {
			t.Fatalf("unexpected user agent: %s", got)
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	body, err := adminRequest(context.Background(), adminOptions{
		registryURL: server.URL,
		adminToken:  "test-token",
	}, http.MethodGet, "/admin/entries", nil)
	if err != nil {
		t.Fatalf("admin request: %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Fatalf("unexpected body: %s", string(body))
	}
}
