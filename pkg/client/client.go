// Package client provides a small HTTP client for public ARD registry surfaces.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ifuryst/ard/pkg/ard"
)

const defaultUserAgent = "ard-go-client/0.1"

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	userAgent  string
	headers    http.Header
}

type Option func(*Client)

func WithHTTPClient(httpClient *http.Client) Option {
	return func(client *Client) {
		if httpClient != nil {
			client.httpClient = httpClient
		}
	}
}

func WithUserAgent(userAgent string) Option {
	return func(client *Client) {
		if strings.TrimSpace(userAgent) != "" {
			client.userAgent = userAgent
		}
	}
}

func WithHeader(name string, value string) Option {
	return func(client *Client) {
		if strings.TrimSpace(name) != "" && value != "" {
			client.headers.Set(name, value)
		}
	}
}

func New(baseURL string, options ...Option) (*Client, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return nil, err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("registry URL must be absolute")
	}
	client := &Client{
		baseURL: parsed,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
		userAgent: defaultUserAgent,
		headers:   http.Header{},
	}
	for _, option := range options {
		option(client)
	}
	return client, nil
}

type BrowseOptions struct {
	PageSize  int
	PageToken string
	Filter    string
	OrderBy   string
}

type HealthResponse struct {
	Status  string `json:"status"`
	Entries int    `json:"entries"`
}

func (client *Client) Search(ctx context.Context, request ard.SearchRequest) (ard.SearchResponse, error) {
	var response ard.SearchResponse
	err := client.doJSON(ctx, http.MethodPost, "/search", nil, request, &response)
	return response, err
}

func (client *Client) Browse(ctx context.Context, options BrowseOptions) (ard.ListResponse, error) {
	query := url.Values{}
	if options.PageSize > 0 {
		query.Set("pageSize", strconv.Itoa(options.PageSize))
	}
	if options.PageToken != "" {
		query.Set("pageToken", options.PageToken)
	}
	if strings.TrimSpace(options.Filter) != "" {
		query.Set("filter", options.Filter)
	}
	if strings.TrimSpace(options.OrderBy) != "" {
		query.Set("orderBy", options.OrderBy)
	}
	var response ard.ListResponse
	err := client.doJSON(ctx, http.MethodGet, "/agents", query, nil, &response)
	return response, err
}

func (client *Client) Explore(ctx context.Context, request ard.ExploreRequest) (ard.ExploreResponse, error) {
	var response ard.ExploreResponse
	err := client.doJSON(ctx, http.MethodPost, "/explore", nil, request, &response)
	return response, err
}

func (client *Client) Catalog(ctx context.Context) (ard.Catalog, error) {
	var response ard.Catalog
	err := client.doJSON(ctx, http.MethodGet, "/.well-known/ai-catalog.json", nil, nil, &response)
	return response, err
}

func (client *Client) Health(ctx context.Context) (HealthResponse, error) {
	var response HealthResponse
	err := client.doJSON(ctx, http.MethodGet, "/health", nil, nil, &response)
	return response, err
}

func (client *Client) doJSON(ctx context.Context, method string, path string, query url.Values, body any, target any) error {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(encoded)
	}

	request, err := http.NewRequestWithContext(ctx, method, client.endpoint(path, query), reader)
	if err != nil {
		return err
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", client.userAgent)
	for name, values := range client.headers {
		for _, value := range values {
			request.Header.Add(name, value)
		}
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return HTTPError{
			Method:     method,
			URL:        request.URL.String(),
			StatusCode: response.StatusCode,
			Body:       raw,
		}
	}
	if target == nil {
		return nil
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return err
	}
	return nil
}

func (client *Client) endpoint(path string, query url.Values) string {
	endpoint := *client.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + path
	endpoint.RawQuery = query.Encode()
	return endpoint.String()
}

type HTTPError struct {
	Method     string
	URL        string
	StatusCode int
	Body       []byte
}

func (err HTTPError) Error() string {
	body := strings.TrimSpace(string(err.Body))
	if body == "" {
		return fmt.Sprintf("%s %s failed with HTTP %d", err.Method, err.URL, err.StatusCode)
	}
	return fmt.Sprintf("%s %s failed with HTTP %d: %s", err.Method, err.URL, err.StatusCode, body)
}
