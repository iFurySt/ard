package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/ifuryst/ard/internal/adapters"
	"github.com/ifuryst/ard/internal/ard"
	"github.com/ifuryst/ard/internal/catalog"
	"github.com/ifuryst/ard/internal/config"
	"github.com/spf13/cobra"
)

type adminOptions struct {
	registryURL string
	adminToken  string
}

func newAdminCommand() *cobra.Command {
	options := adminOptions{}
	command := &cobra.Command{
		Use:   "admin",
		Short: "Manage a remote ARD registry admin API",
	}
	command.PersistentFlags().StringVar(&options.registryURL, "registry-url", envOrDefault("ARD_REGISTRY_URL", "http://127.0.0.1:8080"), "ARD registry base URL")
	command.PersistentFlags().StringVar(&options.adminToken, "admin-token", "", "Admin bearer token. Defaults to ARD_ADMIN_TOKEN.")
	command.AddCommand(newAdminAuditCommand(&options))
	command.AddCommand(newAdminListCommand(&options))
	command.AddCommand(newAdminAddCommand(&options))
	command.AddCommand(newAdminExportCommand(&options))
	command.AddCommand(newAdminRemoveCommand(&options))
	command.AddCommand(newAdminStatusCommand(&options))
	return command
}

func newAdminAuditCommand(options *adminOptions) *cobra.Command {
	var limit int
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "audit",
		Short: "List remote admin audit events",
		RunE: func(cmd *cobra.Command, args []string) error {
			query := url.Values{}
			if limit > 0 {
				query.Set("pageSize", fmt.Sprint(limit))
			}
			body, err := adminRequest(cmd.Context(), *options, http.MethodGet, "/admin/audit?"+query.Encode(), nil)
			if err != nil {
				return err
			}
			if jsonOutput {
				_, err := cmd.OutOrStdout().Write(append(body, '\n'))
				return err
			}
			var response struct {
				Items []storeAuditEvent `json:"items"`
			}
			if err := json.Unmarshal(body, &response); err != nil {
				return err
			}
			for _, event := range response.Items {
				fmt.Fprintf(
					cmd.OutOrStdout(),
					"%-24s  %-14s  %-8s  %s\n",
					event.CreatedAt,
					event.Action,
					event.Status,
					event.Identifier,
				)
			}
			return nil
		},
	}
	command.Flags().IntVar(&limit, "limit", 50, "Maximum audit events to list")
	command.Flags().BoolVar(&jsonOutput, "json", false, "Print remote audit response JSON")
	return command
}

func newAdminListCommand(options *adminOptions) *cobra.Command {
	var kind string
	var status string
	var limit int
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "list",
		Short: "List entries through the remote admin API",
		RunE: func(cmd *cobra.Command, args []string) error {
			query := url.Values{}
			if kind != "" {
				query.Set("kind", kind)
			}
			if status != "" {
				query.Set("status", status)
			}
			if limit > 0 {
				query.Set("pageSize", fmt.Sprint(limit))
			}
			body, err := adminRequest(cmd.Context(), *options, http.MethodGet, "/admin/entries?"+query.Encode(), nil)
			if err != nil {
				return err
			}
			var response ard.ListResponse
			if err := json.Unmarshal(body, &response); err != nil {
				return err
			}
			if jsonOutput {
				_, err := cmd.OutOrStdout().Write(append(body, '\n'))
				return err
			}
			for _, entry := range response.Items {
				fmt.Fprintf(
					cmd.OutOrStdout(),
					"%-52s  %-8s  %-40s  %s\n",
					entry.Identifier,
					entryStatus(entry),
					entry.Type,
					entry.DisplayName,
				)
			}
			return nil
		},
	}
	command.Flags().StringVar(&kind, "kind", "", "Filter by result kind: mcp, a2a, skill, catalog, registry")
	command.Flags().StringVar(&status, "status", "", "Filter by lifecycle status: active, pending, disabled")
	command.Flags().IntVar(&limit, "limit", 20, "Maximum entries to list")
	command.Flags().BoolVar(&jsonOutput, "json", false, "Print remote ListResponse JSON")
	return command
}

func newAdminAddCommand(options *adminOptions) *cobra.Command {
	command := &cobra.Command{
		Use:   "add",
		Short: "Add resources through the remote admin API",
	}
	command.AddCommand(newAdminAddCatalogCommand(options))
	command.AddCommand(newAdminAddArtifactCommand(options, "mcp", "Add an MCP server card through the remote admin API", adapters.LoadMCPServerCard))
	command.AddCommand(newAdminAddArtifactCommand(options, "a2a", "Add an A2A agent card through the remote admin API", adapters.LoadA2AAgentCard))
	command.AddCommand(newAdminAddArtifactCommand(options, "skill", "Add an agent skill through the remote admin API", adapters.LoadSkill))
	command.AddCommand(newAdminAddArtifactCommand(options, "openapi", "Add an OpenAPI document through the remote admin API", adapters.LoadOpenAPI))
	return command
}

func newAdminAddCatalogCommand(options *adminOptions) *cobra.Command {
	command := &cobra.Command{
		Use:   "catalog SOURCE",
		Short: "Add an ai-catalog.json file or URL through the remote admin API",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			loadedCatalog, err := catalog.Load(context.Background(), args[0])
			if err != nil {
				return err
			}
			payload, err := json.Marshal(loadedCatalog)
			if err != nil {
				return err
			}
			if _, err := adminRequest(cmd.Context(), *options, http.MethodPost, "/admin/catalogs", payload); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "remote imported %d catalog entries from %s\n", len(loadedCatalog.Entries), args[0])
			return nil
		},
	}
	return command
}

func newAdminAddArtifactCommand(options *adminOptions, kind string, short string, load artifactLoader) *cobra.Command {
	var adapterOptions adapters.Options
	command := &cobra.Command{
		Use:   kind + " SOURCE",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			entry, err := load(context.Background(), args[0], adapterOptions)
			if err != nil {
				return err
			}
			payload, err := json.Marshal(entry)
			if err != nil {
				return err
			}
			if _, err := adminRequest(cmd.Context(), *options, http.MethodPost, "/admin/entries", payload); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "remote imported %s %s from %s\n", entry.Type, entry.Identifier, args[0])
			return nil
		},
	}
	command.Flags().StringVar(&adapterOptions.Identifier, "identifier", "", "Override generated urn:air identifier")
	command.Flags().StringVar(&adapterOptions.Publisher, "publisher", "", "Override generated publisher domain")
	return command
}

func newAdminExportCommand(options *adminOptions) *cobra.Command {
	command := &cobra.Command{
		Use:   "export",
		Short: "Export remote registry resources through the admin API",
	}
	command.AddCommand(newAdminExportCatalogCommand(options))
	return command
}

func newAdminExportCatalogCommand(options *adminOptions) *cobra.Command {
	var outputPath string
	command := &cobra.Command{
		Use:   "catalog",
		Short: "Export remote registry entries as ai-catalog.json",
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := adminRequest(cmd.Context(), *options, http.MethodGet, "/admin/catalog", nil)
			if err != nil {
				return err
			}
			var exported ard.Catalog
			if err := json.Unmarshal(body, &exported); err != nil {
				return err
			}
			if len(exported.Entries) > 0 {
				if err := ard.ValidateCatalog(exported); err != nil {
					return err
				}
			}
			body = append(bytes.TrimRight(body, "\n"), '\n')
			if outputPath == "" || outputPath == "-" {
				_, err := cmd.OutOrStdout().Write(body)
				return err
			}
			return os.WriteFile(outputPath, body, 0o644)
		},
	}
	command.Flags().StringVarP(&outputPath, "output", "o", "", "Output path, or stdout when omitted")
	return command
}

func newAdminRemoveCommand(options *adminOptions) *cobra.Command {
	var yes bool
	var missingOK bool
	command := &cobra.Command{
		Use:   "remove IDENTIFIER",
		Short: "Remove an entry through the remote admin API",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			identifier := args[0]
			if err := ard.ValidateIdentifier(identifier); err != nil {
				return err
			}
			if !yes {
				return fmt.Errorf("refusing to remove %s without --yes", identifier)
			}
			_, err := adminRequest(cmd.Context(), *options, http.MethodDelete, "/admin/entries/"+url.PathEscape(identifier), nil)
			if err != nil {
				if missingOK && strings.Contains(err.Error(), "HTTP 404") {
					fmt.Fprintf(cmd.OutOrStdout(), "remote entry not found: %s\n", identifier)
					return nil
				}
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "remote removed %s\n", identifier)
			return nil
		},
	}
	command.Flags().BoolVar(&yes, "yes", false, "Confirm removal")
	command.Flags().BoolVar(&missingOK, "missing-ok", false, "Treat a missing identifier as success")
	return command
}

func newAdminStatusCommand(options *adminOptions) *cobra.Command {
	command := &cobra.Command{
		Use:   "status IDENTIFIER STATUS",
		Short: "Set a remote entry lifecycle status",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			identifier := args[0]
			if err := ard.ValidateIdentifier(identifier); err != nil {
				return err
			}
			status := strings.ToLower(strings.TrimSpace(args[1]))
			switch status {
			case "active", "pending", "disabled":
			default:
				return fmt.Errorf("status must be one of: active, pending, disabled")
			}
			payload, err := json.Marshal(map[string]string{"status": status})
			if err != nil {
				return err
			}
			if _, err := adminRequest(cmd.Context(), *options, http.MethodPatch, "/admin/entries/"+url.PathEscape(identifier)+"/status", payload); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "remote set %s status to %s\n", identifier, status)
			return nil
		},
	}
	return command
}

func entryStatus(entry ard.CatalogEntry) string {
	if entry.Metadata == nil {
		return ""
	}
	status, ok := entry.Metadata["ard.status"].(string)
	if !ok {
		return ""
	}
	return status
}

type storeAuditEvent struct {
	Action     string `json:"action"`
	Identifier string `json:"identifier,omitempty"`
	Status     string `json:"status,omitempty"`
	CreatedAt  string `json:"createdAt"`
}

func adminRequest(ctx context.Context, options adminOptions, method string, path string, payload []byte) ([]byte, error) {
	token := config.AdminToken(options.adminToken)
	if token == "" {
		return nil, fmt.Errorf("admin token is required; pass --admin-token or set ARD_ADMIN_TOKEN")
	}
	baseURL := strings.TrimRight(options.registryURL, "/")
	request, err := http.NewRequestWithContext(ctx, method, baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "ardctl/0.1")
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	client := http.Client{Timeout: 20 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, fmt.Errorf("admin API request failed with HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}
