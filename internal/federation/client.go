package federation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ifuryst/ard/internal/ard"
	"github.com/ifuryst/ard/internal/requestid"
	"github.com/ifuryst/ard/internal/tracecontext"
)

const MaxUpstreamRegistries = 3
const MaxResponseBytes = 2 * 1024 * 1024

type Client struct {
	httpClient http.Client
}

type SearchPage struct {
	Results        []ard.SearchResult
	NextPageTokens map[string]string
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return requestid.With(ctx, requestID)
}

func NewClient() Client {
	return Client{httpClient: http.Client{Timeout: 10 * time.Second}}
}

func (client Client) Search(ctx context.Context, referrals []ard.CatalogEntry, request ard.SearchRequest) []ard.SearchResult {
	return client.SearchPage(ctx, referrals, request, nil).Results
}

func (client Client) SearchPage(ctx context.Context, referrals []ard.CatalogEntry, request ard.SearchRequest, pageTokens map[string]string) SearchPage {
	request.Federation = "none"
	results := []ard.SearchResult{}
	nextPageTokens := map[string]string{}
	for index, referral := range referrals {
		if index >= MaxUpstreamRegistries {
			break
		}
		key := ReferralKey(referral)
		request.PageToken = ""
		if pageTokens != nil {
			pageToken, ok := pageTokens[key]
			if !ok {
				continue
			}
			request.PageToken = pageToken
		}
		upstreamPage, err := client.searchOne(ctx, referral, request)
		if err != nil {
			continue
		}
		results = append(results, upstreamPage.Results...)
		if upstreamPage.PageToken != "" {
			nextPageTokens[key] = upstreamPage.PageToken
		}
	}
	return SearchPage{Results: results, NextPageTokens: nextPageTokens}
}

func ReferralKey(referral ard.CatalogEntry) string {
	if strings.TrimSpace(referral.Identifier) != "" {
		return referral.Identifier
	}
	return referral.URL
}

func (client Client) searchOne(ctx context.Context, referral ard.CatalogEntry, request ard.SearchRequest) (ard.SearchResponse, error) {
	endpoint, err := searchEndpoint(referral.URL)
	if err != nil {
		return ard.SearchResponse{}, err
	}
	body, err := json.Marshal(request)
	if err != nil {
		return ard.SearchResponse{}, err
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return ard.SearchResponse{}, err
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("User-Agent", "ard/0.1")
	requestid.SetHeader(httpRequest.Header, ctx)
	tracecontext.SetHeader(httpRequest.Header, ctx)

	response, err := client.httpClient.Do(httpRequest)
	if err != nil {
		return ard.SearchResponse{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return ard.SearchResponse{}, nil
	}
	var parsed ard.SearchResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, MaxResponseBytes)).Decode(&parsed); err != nil {
		return ard.SearchResponse{}, err
	}
	for index := range parsed.Results {
		if parsed.Results[index].Source == "" {
			parsed.Results[index].Source = referral.URL
		}
	}
	return parsed, nil
}

func searchEndpoint(value string) (string, error) {
	parsed, err := url.Parse(value)
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("registry referral URL must be absolute")
	}
	if strings.HasSuffix(strings.TrimRight(parsed.Path, "/"), "/search") {
		return parsed.String(), nil
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/search"
	return parsed.String(), nil
}
