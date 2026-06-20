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
)

const MaxUpstreamRegistries = 3
const MaxResponseBytes = 2 * 1024 * 1024

type Client struct {
	httpClient http.Client
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return requestid.With(ctx, requestID)
}

func NewClient() Client {
	return Client{httpClient: http.Client{Timeout: 10 * time.Second}}
}

func (client Client) Search(ctx context.Context, referrals []ard.CatalogEntry, request ard.SearchRequest) []ard.SearchResult {
	request.Federation = "none"
	request.PageToken = ""
	results := []ard.SearchResult{}
	for index, referral := range referrals {
		if index >= MaxUpstreamRegistries {
			break
		}
		upstreamResults, err := client.searchOne(ctx, referral, request)
		if err != nil {
			continue
		}
		results = append(results, upstreamResults...)
	}
	return results
}

func (client Client) searchOne(ctx context.Context, referral ard.CatalogEntry, request ard.SearchRequest) ([]ard.SearchResult, error) {
	endpoint, err := searchEndpoint(referral.URL)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("User-Agent", "ard/0.1")
	requestid.SetHeader(httpRequest.Header, ctx)

	response, err := client.httpClient.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, nil
	}
	var parsed ard.SearchResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, MaxResponseBytes)).Decode(&parsed); err != nil {
		return nil, err
	}
	for index := range parsed.Results {
		if parsed.Results[index].Source == "" {
			parsed.Results[index].Source = referral.URL
		}
	}
	return parsed.Results, nil
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
