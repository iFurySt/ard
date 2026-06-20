package verify

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ifuryst/ard/internal/ard"
	"github.com/ifuryst/ard/internal/requestid"
)

const maxVerifyArtifactBytes = 4 << 20

type SourceDigestResult struct {
	Identifier string `json:"identifier"`
	URL        string `json:"url"`
	Expected   string `json:"expected"`
	Actual     string `json:"actual"`
	Verified   bool   `json:"verified"`
}

func VerifySourceDigests(ctx context.Context, catalog ard.Catalog) ([]SourceDigestResult, error) {
	results := []SourceDigestResult{}
	for _, entry := range catalog.Entries {
		expected := trustString(entry.TrustManifest, "sourceDigest")
		if expected == "" {
			continue
		}
		if entry.URL == "" {
			return results, fmt.Errorf("%s: sourceDigest verification requires url delivery", entry.Identifier)
		}
		actual, err := fetchDigest(ctx, entry.URL)
		if err != nil {
			return results, fmt.Errorf("%s: verify sourceDigest: %w", entry.Identifier, err)
		}
		result := SourceDigestResult{
			Identifier: entry.Identifier,
			URL:        entry.URL,
			Expected:   expected,
			Actual:     actual,
			Verified:   strings.EqualFold(expected, actual),
		}
		results = append(results, result)
		if !result.Verified {
			return results, fmt.Errorf("%s: sourceDigest mismatch: expected %s, got %s", entry.Identifier, expected, actual)
		}
	}
	return results, nil
}

func fetchDigest(ctx context.Context, artifactURL string) (string, error) {
	client := http.Client{Timeout: 20 * time.Second}
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, artifactURL, nil)
		if err != nil {
			return "", err
		}
		request.Header.Set("Accept", "*/*")
		request.Header.Set("User-Agent", "ard/0.1")
		requestid.SetHeader(request.Header, ctx)
		response, err := client.Do(request)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt+1) * time.Second)
			continue
		}
		digest, readErr := func() (string, error) {
			defer response.Body.Close()
			if response.StatusCode < 200 || response.StatusCode > 299 {
				return "", fmt.Errorf("artifact request failed with HTTP %d", response.StatusCode)
			}
			return readerSHA256(response.Body)
		}()
		if readErr != nil {
			lastErr = readErr
			if response.StatusCode >= 400 && response.StatusCode < 500 {
				return "", readErr
			}
			time.Sleep(time.Duration(attempt+1) * time.Second)
			continue
		}
		return digest, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("artifact request failed")
	}
	return "", lastErr
}

func readerSHA256(reader io.Reader) (string, error) {
	limited := io.LimitReader(reader, maxVerifyArtifactBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return "", err
	}
	if len(data) > maxVerifyArtifactBytes {
		return "", fmt.Errorf("artifact exceeds %d bytes", maxVerifyArtifactBytes)
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func trustString(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, _ := values[key].(string)
	return strings.TrimSpace(value)
}
