package cli

import (
	"context"
	"fmt"

	"github.com/ifuryst/ard/internal/ard"
	"github.com/ifuryst/ard/internal/config"
	"github.com/ifuryst/ard/internal/store"
	"github.com/spf13/cobra"
)

func newRemoveCommand(root *rootOptions) *cobra.Command {
	var yes bool
	var missingOK bool
	command := &cobra.Command{
		Use:   "remove IDENTIFIER",
		Short: "Remove a registry entry from local storage",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			identifier := args[0]
			if err := ard.ValidateIdentifier(identifier); err != nil {
				return err
			}
			if !yes {
				return fmt.Errorf("refusing to remove %s without --yes", identifier)
			}

			ctx := context.Background()
			registryStore, err := store.Open(config.DatabaseURL(root.databaseURL))
			if err != nil {
				return err
			}
			defer registryStore.Close()
			if err := registryStore.AutoMigrate(); err != nil {
				return err
			}

			removed, err := registryStore.DeleteEntry(ctx, identifier)
			if err != nil {
				return err
			}
			if !removed {
				if missingOK {
					fmt.Fprintf(cmd.OutOrStdout(), "entry not found: %s\n", identifier)
					return nil
				}
				return fmt.Errorf("entry not found: %s", identifier)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "removed %s\n", identifier)
			return nil
		},
	}
	command.Flags().BoolVar(&yes, "yes", false, "Confirm removal")
	command.Flags().BoolVar(&missingOK, "missing-ok", false, "Treat a missing identifier as success")
	return command
}
