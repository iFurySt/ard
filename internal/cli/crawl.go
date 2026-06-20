package cli

import (
	"context"
	"fmt"

	"github.com/ifuryst/ard/internal/catalog"
	"github.com/ifuryst/ard/internal/config"
	"github.com/ifuryst/ard/internal/store"
	"github.com/spf13/cobra"
)

func newCrawlCommand(root *rootOptions) *cobra.Command {
	command := &cobra.Command{
		Use:   "crawl URL",
		Short: "Discover and import a site's well-known AI catalog",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			catalogURL, err := catalog.WellKnownCatalogURL(args[0])
			if err != nil {
				return err
			}
			loadedCatalog, err := catalog.Load(ctx, catalogURL)
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
			statuses, err := evaluatePolicy(root, loadedCatalog)
			if err != nil {
				return err
			}
			if err := registryStore.UpsertCatalogWithStatuses(ctx, loadedCatalog, catalogURL, statuses); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), store.FormatCatalogImport(len(loadedCatalog.Entries), catalogURL))
			return nil
		},
	}
	return command
}
