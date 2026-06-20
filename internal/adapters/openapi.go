package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/ifuryst/ard/internal/ard"
	"gopkg.in/yaml.v3"
)

type openAPIDocument struct {
	OpenAPI string                    `json:"openapi" yaml:"openapi"`
	Swagger string                    `json:"swagger" yaml:"swagger"`
	Info    openAPIInfo               `json:"info" yaml:"info"`
	Servers []openAPIServer           `json:"servers" yaml:"servers"`
	Paths   map[string]map[string]any `json:"paths" yaml:"paths"`
	Tags    []openAPITag              `json:"tags" yaml:"tags"`
	Raw     map[string]any            `json:"-" yaml:"-"`
}

type openAPIInfo struct {
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`
	Version     string `json:"version" yaml:"version"`
}

type openAPIServer struct {
	URL string `json:"url" yaml:"url"`
}

type openAPITag struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description" yaml:"description"`
}

func LoadOpenAPI(ctx context.Context, source string, options Options) (ard.CatalogEntry, error) {
	artifact, err := readSource(ctx, source, "application/openapi+json, application/openapi+yaml, application/vnd.oai.openapi+json, application/vnd.oai.openapi, application/json, application/yaml, text/yaml")
	if err != nil {
		return ard.CatalogEntry{}, err
	}

	var raw map[string]any
	if err := yaml.Unmarshal(artifact.Data, &raw); err != nil {
		return ard.CatalogEntry{}, fmt.Errorf("parse OpenAPI document: %w", err)
	}
	var document openAPIDocument
	if err := yaml.Unmarshal(artifact.Data, &document); err != nil {
		return ard.CatalogEntry{}, fmt.Errorf("parse OpenAPI document: %w", err)
	}
	document.Raw = raw
	if document.OpenAPI == "" && document.Swagger == "" {
		return ard.CatalogEntry{}, fmt.Errorf("OpenAPI document must include openapi or swagger version")
	}
	if document.Info.Title == "" {
		return ard.CatalogEntry{}, fmt.Errorf("OpenAPI document info.title is required")
	}

	displayName := document.Info.Title
	identifier, err := identifierFor(source, "api", displayName, options)
	if err != nil {
		return ard.CatalogEntry{}, err
	}
	entry := ard.CatalogEntry{
		Identifier:            identifier,
		DisplayName:           displayName,
		Type:                  ard.TypeOpenAPI,
		Description:           document.Info.Description,
		Version:               document.Info.Version,
		Tags:                  openAPITags(document),
		Capabilities:          openAPICapabilities(document),
		RepresentativeQueries: openAPIRepresentativeQueries(document),
		Metadata: map[string]any{
			"adapter":   "openapi",
			"apiFormat": firstNonEmpty(document.OpenAPI, "swagger "+document.Swagger),
			"pathCount": len(document.Paths),
		},
	}
	if len(document.Servers) > 0 && document.Servers[0].URL != "" {
		entry.Metadata["serverUrl"] = document.Servers[0].URL
	}
	if artifact.IsURL {
		entry.URL = source
	} else {
		entry.Data = raw
	}
	if err := ard.ValidateCatalogEntry(entry); err != nil {
		return ard.CatalogEntry{}, err
	}
	return entry, nil
}

func openAPITags(document openAPIDocument) []string {
	values := []string{"openapi", "api"}
	for _, tag := range document.Tags {
		values = append(values, tag.Name)
	}
	for _, pathItem := range document.Paths {
		for _, operation := range pathItem {
			for _, tag := range stringSliceFromMap(operationMap(operation), "tags") {
				values = append(values, tag)
			}
		}
	}
	return uniqueStrings(values)
}

func openAPICapabilities(document openAPIDocument) []string {
	values := make([]string, 0, len(document.Paths)*2)
	paths := sortedOpenAPIPaths(document.Paths)
	for _, path := range paths {
		pathItem := document.Paths[path]
		methods := sortedOpenAPIMethods(pathItem)
		for _, method := range methods {
			operation := operationMap(pathItem[method])
			if operationID, ok := operation["operationId"].(string); ok {
				values = append(values, operationID)
				continue
			}
			values = append(values, strings.ToUpper(method)+" "+path)
		}
	}
	return uniqueStrings(values)
}

func openAPIRepresentativeQueries(document openAPIDocument) []string {
	values := []string{}
	for _, capability := range openAPICapabilities(document) {
		values = append(values, "use "+capability)
		if len(values) == 5 {
			break
		}
	}
	if len(values) == 1 {
		values = append(values, "call "+document.Info.Title)
	}
	if len(values) < 2 {
		return nil
	}
	return values
}

func sortedOpenAPIPaths(paths map[string]map[string]any) []string {
	values := make([]string, 0, len(paths))
	for path := range paths {
		values = append(values, path)
	}
	sort.Strings(values)
	return values
}

func sortedOpenAPIMethods(pathItem map[string]any) []string {
	allowed := map[string]struct{}{
		"delete": {}, "get": {}, "head": {}, "options": {}, "patch": {}, "post": {}, "put": {}, "trace": {},
	}
	values := make([]string, 0, len(pathItem))
	for method := range pathItem {
		method = strings.ToLower(method)
		if _, ok := allowed[method]; ok {
			values = append(values, method)
		}
	}
	sort.Strings(values)
	return values
}

func operationMap(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		return typed
	case map[any]any:
		result := map[string]any{}
		for key, value := range typed {
			if keyString, ok := key.(string); ok {
				result[keyString] = value
			}
		}
		return result
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return nil
		}
		var result map[string]any
		if err := json.Unmarshal(data, &result); err != nil {
			return nil
		}
		return result
	}
}

func stringSliceFromMap(values map[string]any, key string) []string {
	raw, ok := values[key]
	if !ok {
		return nil
	}
	switch typed := raw.(type) {
	case []string:
		return typed
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if value, ok := item.(string); ok {
				result = append(result, value)
			}
		}
		return result
	default:
		return nil
	}
}
