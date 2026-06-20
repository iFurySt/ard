package cli

import (
	"context"
	"fmt"

	"github.com/ifuryst/ard/internal/adapters"
	"github.com/ifuryst/ard/internal/ard"
	"github.com/ifuryst/ard/internal/catalog"
	"github.com/ifuryst/ard/internal/config"
	"github.com/ifuryst/ard/internal/store"
	"github.com/spf13/cobra"
)

func newAddCommand(root *rootOptions) *cobra.Command {
	command := &cobra.Command{
		Use:   "add",
		Short: "Add agentic resources to the registry",
	}
	command.AddCommand(newAddCatalogCommand(root))
	command.AddCommand(newAddArtifactCommand(root, "mcp", "Import an MCP server card", adapters.LoadMCPServerCard))
	command.AddCommand(newAddArtifactCommand(root, "a2a", "Import an A2A agent card", adapters.LoadA2AAgentCard))
	command.AddCommand(newAddArtifactCommand(root, "skill", "Import an agent skill", adapters.LoadSkill))
	command.AddCommand(newAddArtifactCommand(root, "openapi", "Import an OpenAPI document", adapters.LoadOpenAPI))
	return command
}

func newAddCatalogCommand(root *rootOptions) *cobra.Command {
	command := &cobra.Command{
		Use:   "catalog SOURCE",
		Short: "Import an ai-catalog.json file or URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			source := args[0]
			loadedCatalog, err := catalog.Load(ctx, source)
			if err != nil {
				return err
			}

			registryStore, err := store.Open(config.DatabaseURL(root.databaseURL))
			if err != nil {
				return err
			}
			defer registryStore.Close()
			if err := registryStore.AutoMigrate(); err != nil {
				return err
			}
			if err := registryStore.UpsertCatalog(ctx, loadedCatalog, source); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), store.FormatCatalogImport(len(loadedCatalog.Entries), source))
			return nil
		},
	}
	return command
}

type artifactLoader func(context.Context, string, adapters.Options) (ard.CatalogEntry, error)

func newAddArtifactCommand(root *rootOptions, kind string, short string, load artifactLoader) *cobra.Command {
	var options adapters.Options
	command := &cobra.Command{
		Use:   kind + " SOURCE",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			source := args[0]
			entry, err := load(ctx, source, options)
			if err != nil {
				return err
			}

			registryStore, err := store.Open(config.DatabaseURL(root.databaseURL))
			if err != nil {
				return err
			}
			defer registryStore.Close()
			if err := registryStore.AutoMigrate(); err != nil {
				return err
			}
			if err := registryStore.UpsertCatalog(ctx, adapters.CatalogFromEntry(entry), source); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), adapters.FormatArtifactImport(entry, source))
			return nil
		},
	}
	command.Flags().StringVar(&options.Identifier, "identifier", "", "Override generated urn:air identifier")
	command.Flags().StringVar(&options.Publisher, "publisher", "", "Override generated publisher domain")
	return command
}
