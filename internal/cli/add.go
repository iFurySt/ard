package cli

import (
	"context"
	"fmt"

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
