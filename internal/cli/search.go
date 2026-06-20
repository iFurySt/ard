package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/ifuryst/ard/internal/ard"
	"github.com/spf13/cobra"
)

func newSearchCommand() *cobra.Command {
	var registryURL string
	var kind string
	var jsonOutput bool
	var limit int
	command := &cobra.Command{
		Use:   "search QUERY",
		Short: "Search an ARD registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filter := ard.Filter{}
			if kind != "" {
				filter["type"] = []string{mediaTypeForKind(kind)}
			}
			response, raw, err := searchRegistry(registryURL, ard.SearchRequest{
				Query: ard.SearchQuery{
					Text:   args[0],
					Filter: filter,
				},
				Federation: "none",
				PageSize:   limit,
			})
			if err != nil {
				return err
			}
			if jsonOutput {
				_, err := cmd.OutOrStdout().Write(raw)
				if err == nil {
					fmt.Fprintln(cmd.OutOrStdout())
				}
				return err
			}
			for _, result := range response.Results {
				fmt.Fprintf(
					cmd.OutOrStdout(),
					"%3d  %-36s  %s\n",
					result.Score,
					result.Type,
					result.DisplayName,
				)
			}
			return nil
		},
	}
	command.Flags().StringVar(&registryURL, "registry-url", envOrDefault("ARD_REGISTRY_URL", "http://127.0.0.1:8080"), "ARD registry base URL")
	command.Flags().StringVar(&kind, "kind", "", "Filter by result kind: mcp, a2a, skill, catalog, registry")
	command.Flags().BoolVar(&jsonOutput, "json", false, "Print raw ARD SearchResponse JSON")
	command.Flags().IntVar(&limit, "limit", 10, "Maximum search results")
	return command
}

func searchRegistry(registryURL string, request ard.SearchRequest) (ard.SearchResponse, []byte, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return ard.SearchResponse{}, nil, err
	}
	client := http.Client{Timeout: 20 * time.Second}
	response, err := client.Post(registryURL+"/search", "application/json", bytes.NewReader(body))
	if err != nil {
		return ard.SearchResponse{}, nil, err
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(response.Body)
	if err != nil {
		return ard.SearchResponse{}, nil, err
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return ard.SearchResponse{}, raw, fmt.Errorf("registry search failed with HTTP %d: %s", response.StatusCode, string(raw))
	}

	var parsed ard.SearchResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return ard.SearchResponse{}, raw, err
	}
	return parsed, raw, nil
}

func mediaTypeForKind(kind string) string {
	switch kind {
	case "mcp":
		return ard.TypeMCPServerCard
	case "a2a":
		return ard.TypeA2AAgentCard
	case "skill":
		return ard.TypeAISkill
	case "catalog":
		return ard.TypeAICatalog
	case "registry":
		return ard.TypeAIRegistry
	default:
		return kind
	}
}

func envOrDefault(name string, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
