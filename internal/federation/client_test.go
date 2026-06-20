package federation

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ifuryst/ard/internal/ard"
)

func TestClientSearchForcesNonRecursiveFederation(t *testing.T) {
	var seenRequest ard.SearchRequest
	seenRequestID := ""
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/search" {
			http.Error(response, "unexpected path", http.StatusNotFound)
			return
		}
		if request.Header.Get("Authorization") != "" {
			http.Error(response, "authorization header leaked", http.StatusBadRequest)
			return
		}
		seenRequestID = request.Header.Get("X-Request-ID")
		if err := json.NewDecoder(request.Body).Decode(&seenRequest); err != nil {
			http.Error(response, err.Error(), http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(response).Encode(ard.SearchResponse{
			Results: []ard.SearchResult{
				{
					CatalogEntry: ard.CatalogEntry{
						Identifier:  "urn:air:upstream.example.com:server:weather",
						DisplayName: "Upstream Weather",
						Type:        ard.TypeMCPServerCard,
						URL:         "https://upstream.example.com/weather.json",
					},
					Score: 80,
				},
			},
		})
	}))
	t.Cleanup(server.Close)

	ctx := WithRequestID(t.Context(), "federation-unit-request")
	results := NewClient().Search(ctx, []ard.CatalogEntry{
		{
			Identifier: "urn:air:upstream.example.com:registry:public",
			Type:       ard.TypeAIRegistry,
			URL:        server.URL,
		},
	}, ard.SearchRequest{
		Query:      ard.SearchQuery{Text: "weather"},
		Federation: "auto",
		PageSize:   5,
	})

	if seenRequest.Federation != "none" {
		t.Fatalf("expected upstream federation none, got %q", seenRequest.Federation)
	}
	if seenRequestID != "federation-unit-request" {
		t.Fatalf("expected request ID propagation, got %q", seenRequestID)
	}
	if len(results) != 1 {
		t.Fatalf("expected one upstream result, got %#v", results)
	}
	if results[0].Source != server.URL {
		t.Fatalf("expected source defaulted to referral URL, got %q", results[0].Source)
	}
}

func TestSearchEndpoint(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "base URL",
			in:   "https://registry.example.com",
			want: "https://registry.example.com/search",
		},
		{
			name: "existing search URL",
			in:   "https://registry.example.com/api/search",
			want: "https://registry.example.com/api/search",
		},
		{
			name: "nested base URL",
			in:   "https://registry.example.com/api/v1/",
			want: "https://registry.example.com/api/v1/search",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := searchEndpoint(test.in)
			if err != nil {
				t.Fatalf("searchEndpoint returned error: %v", err)
			}
			if got != test.want {
				t.Fatalf("searchEndpoint(%q) = %q, want %q", test.in, got, test.want)
			}
		})
	}
}

func TestSearchEndpointRejectsRelativeURL(t *testing.T) {
	if _, err := searchEndpoint("/search"); err == nil {
		t.Fatal("expected relative URL to fail")
	}
}
