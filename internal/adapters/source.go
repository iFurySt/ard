package adapters

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/ifuryst/ard/internal/ard"
	"github.com/ifuryst/ard/internal/requestid"
)

const maxArtifactBytes = 4 << 20

type Options struct {
	Identifier      string
	Publisher       string
	PinSourceDigest bool
}

type artifactSource struct {
	Data         []byte
	IsURL        bool
	SourceDigest string
}

func readSource(ctx context.Context, source string, accept string) (artifactSource, error) {
	if isHTTPURL(source) {
		client := http.Client{Timeout: 20 * time.Second}
		var lastErr error
		for attempt := 0; attempt < 3; attempt++ {
			request, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
			if err != nil {
				return artifactSource{}, err
			}
			request.Header.Set("Accept", accept)
			request.Header.Set("User-Agent", "ard/0.1")
			requestid.SetHeader(request.Header, ctx)

			response, err := client.Do(request)
			if err != nil {
				lastErr = err
				time.Sleep(time.Duration(attempt+1) * time.Second)
				continue
			}
			data, readErr := func() ([]byte, error) {
				defer response.Body.Close()
				if response.StatusCode < 200 || response.StatusCode > 299 {
					return nil, fmt.Errorf("artifact request failed with HTTP %d", response.StatusCode)
				}
				return readLimited(response.Body)
			}()
			if readErr != nil {
				lastErr = readErr
				if response.StatusCode >= 400 && response.StatusCode < 500 {
					return artifactSource{}, readErr
				}
				time.Sleep(time.Duration(attempt+1) * time.Second)
				continue
			}
			return artifactSource{Data: data, IsURL: true, SourceDigest: sha256Digest(data)}, nil
		}
		if lastErr == nil {
			lastErr = fmt.Errorf("artifact request failed")
		}
		return artifactSource{}, lastErr
	}

	file, err := os.Open(source)
	if err != nil {
		return artifactSource{}, err
	}
	defer file.Close()
	data, err := readLimited(file)
	return artifactSource{Data: data, SourceDigest: sha256Digest(data)}, err
}

func requireURLForSourceDigest(source string, artifact artifactSource, options Options) error {
	if options.PinSourceDigest && !artifact.IsURL {
		return fmt.Errorf("--pin-source-digest requires URL source, got %s", source)
	}
	return nil
}

func readLimited(reader io.Reader) ([]byte, error) {
	limited := io.LimitReader(reader, maxArtifactBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(data) > maxArtifactBytes {
		return nil, fmt.Errorf("artifact exceeds %d bytes", maxArtifactBytes)
	}
	return data, nil
}

func isHTTPURL(source string) bool {
	return strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://")
}

func identifierFor(source string, namespace string, displayName string, options Options) (string, error) {
	if options.Identifier != "" {
		return options.Identifier, nil
	}
	publisher := strings.TrimSpace(options.Publisher)
	if publisher == "" {
		publisher = publisherFromSource(source)
	}
	name := slugify(displayName)
	if name == "" {
		name = namespace
	}
	return fmt.Sprintf("urn:air:%s:%s:%s", publisher, namespace, name), nil
}

func applySourceDigestTrust(entry *ard.CatalogEntry, digest string) {
	if digest == "" {
		return
	}
	if entry.TrustManifest == nil {
		entry.TrustManifest = map[string]any{}
	}
	if _, ok := entry.TrustManifest["identity"]; !ok {
		entry.TrustManifest["identity"] = "https://" + ard.Publisher(entry.Identifier)
	}
	entry.TrustManifest["sourceDigest"] = digest
}

func sha256Digest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func publisherFromSource(source string) string {
	if !isHTTPURL(source) {
		return "agent.localhost"
	}
	parsed, err := url.Parse(source)
	if err != nil || parsed.Host == "" {
		return "agent.localhost"
	}
	host := parsed.Hostname()
	if host == "" {
		host, _, _ = net.SplitHostPort(parsed.Host)
	}
	host = strings.ToLower(host)
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return "agent.localhost"
	}
	return host
}

var nonSlug = regexp.MustCompile(`[^a-z0-9._-]+`)

func slugify(value string) string {
	lower := strings.ToLower(strings.TrimSpace(value))
	lower = strings.ReplaceAll(lower, "/", "-")
	lower = strings.ReplaceAll(lower, "\\", "-")
	lower = nonSlug.ReplaceAllString(lower, "-")
	lower = strings.Trim(lower, "-._")
	for strings.Contains(lower, "--") {
		lower = strings.ReplaceAll(lower, "--", "-")
	}
	return lower
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
