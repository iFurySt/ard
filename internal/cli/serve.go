package cli

import (
	"fmt"

	"github.com/ifuryst/ard/internal/config"
	"github.com/ifuryst/ard/internal/httpapi"
	"github.com/ifuryst/ard/internal/store"
	"github.com/spf13/cobra"
)

func newServeCommand(root *rootOptions) *cobra.Command {
	var addr string
	command := &cobra.Command{
		Use:   "serve",
		Short: "Run the ARD registry server",
		RunE: func(cmd *cobra.Command, args []string) error {
			registryStore, err := store.Open(config.DatabaseURL(root.databaseURL))
			if err != nil {
				return err
			}
			defer registryStore.Close()
			if err := registryStore.AutoMigrate(); err != nil {
				return err
			}

			router := httpapi.NewRouter(registryStore)
			fmt.Fprintf(cmd.ErrOrStderr(), "listening on %s\n", addr)
			return router.Run(addr)
		},
	}
	command.Flags().StringVar(&addr, "addr", ":8080", "HTTP listen address")
	return command
}
