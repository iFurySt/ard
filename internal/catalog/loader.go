package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ifuryst/ard/internal/ard"
)

const maxCatalogBytes = 4 << 20

func Load(ctx context.Context, source string) (ard.Catalog, error) {
	var reader io.ReadCloser
	var err error
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		reader, err = loadHTTP(ctx, source)
	} else {
		reader, err = os.Open(source)
	}
	if err != nil {
		return ard.Catalog{}, err
	}
	defer reader.Close()

	limited := io.LimitReader(reader, maxCatalogBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return ard.Catalog{}, err
	}
	if len(data) > maxCatalogBytes {
		return ard.Catalog{}, fmt.Errorf("catalog exceeds %d bytes", maxCatalogBytes)
	}

	var catalog ard.Catalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return ard.Catalog{}, err
	}
	if err := ard.ValidateCatalog(catalog); err != nil {
		return ard.Catalog{}, err
	}
	return catalog, nil
}

func loadHTTP(ctx context.Context, source string) (io.ReadCloser, error) {
	client := http.Client{Timeout: 20 * time.Second}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "ard/0.1")

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		response.Body.Close()
		return nil, fmt.Errorf("catalog request failed with HTTP %d", response.StatusCode)
	}
	return response.Body, nil
}
