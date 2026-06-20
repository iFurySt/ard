package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ifuryst/ard/internal/ard"
	"github.com/ifuryst/ard/internal/config"
	"github.com/ifuryst/ard/internal/store"
	"github.com/spf13/cobra"
)

func newListCommand(root *rootOptions) *cobra.Command {
	var kind string
	var limit int
	var pageToken string
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "list",
		Short: "List registry entries from local storage",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			registryStore, err := store.Open(config.DatabaseURL(root.databaseURL))
			if err != nil {
				return err
			}
			defer registryStore.Close()
			if err := registryStore.AutoMigrate(); err != nil {
				return err
			}

			page, err := registryStore.ListEntriesPage(ctx, store.ListOptions{
				Limit:     limit,
				PageToken: pageToken,
				Type:      mediaTypeForKind(kind),
			})
			if err != nil {
				return err
			}
			if jsonOutput {
				payload, err := json.MarshalIndent(ard.ListResponse{Items: page.Entries, Total: int(page.Total), PageToken: page.NextPageToken}, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(payload))
				return nil
			}
			for _, entry := range page.Entries {
				fmt.Fprintf(cmd.OutOrStdout(), "%-52s  %-40s  %s\n", entry.Identifier, entry.Type, entry.DisplayName)
			}
			return nil
		},
	}
	command.Flags().StringVar(&kind, "kind", "", "Filter by result kind: mcp, a2a, skill, catalog, registry")
	command.Flags().IntVar(&limit, "limit", 20, "Maximum entries to list")
	command.Flags().StringVar(&pageToken, "page-token", "", "Opaque page token returned by a previous list response")
	command.Flags().BoolVar(&jsonOutput, "json", false, "Print machine-readable list response JSON")
	return command
}
