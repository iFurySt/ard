package adapters

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

const maxArtifactBytes = 4 << 20

type Options struct {
	Identifier string
	Publisher  string
}

type artifactSource struct {
	Data  []byte
	IsURL bool
}

func readSource(ctx context.Context, source string, accept string) (artifactSource, error) {
	if isHTTPURL(source) {
		client := http.Client{Timeout: 20 * time.Second}
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
		if err != nil {
			return artifactSource{}, err
		}
		request.Header.Set("Accept", accept)
		request.Header.Set("User-Agent", "ard/0.1")

		response, err := client.Do(request)
		if err != nil {
			return artifactSource{}, err
		}
		defer response.Body.Close()
		if response.StatusCode < 200 || response.StatusCode > 299 {
			return artifactSource{}, fmt.Errorf("artifact request failed with HTTP %d", response.StatusCode)
		}
		data, err := readLimited(response.Body)
		return artifactSource{Data: data, IsURL: true}, err
	}

	file, err := os.Open(source)
	if err != nil {
		return artifactSource{}, err
	}
	defer file.Close()
	data, err := readLimited(file)
	return artifactSource{Data: data}, err
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
