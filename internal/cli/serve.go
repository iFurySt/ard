package cli

import (
	"fmt"

	"github.com/ifuryst/ard/internal/config"
	"github.com/ifuryst/ard/internal/httpapi"
	"github.com/ifuryst/ard/internal/policy"
	"github.com/ifuryst/ard/internal/store"
	"github.com/spf13/cobra"
)

func newServeCommand(root *rootOptions) *cobra.Command {
	var addr string
	command := &cobra.Command{
		Use:   "serve",
		Short: "Run the ARD registry server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServer(cmd, root, addr)
		},
	}
	command.Flags().StringVar(&addr, "addr", ":8080", "HTTP listen address")
	addAdminTokenFlag(command, root)
	return command
}

func runServer(cmd *cobra.Command, root *rootOptions, addr string) error {
	registryStore, err := store.Open(config.DatabaseURL(root.databaseURL))
	if err != nil {
		return err
	}
	defer registryStore.Close()
	if err := registryStore.AutoMigrate(); err != nil {
		return err
	}

	var loadedPolicy *policy.Policy
	if policyFile := config.PolicyFile(root.policyFile); policyFile != "" {
		parsedPolicy, err := policy.LoadFile(policyFile)
		if err != nil {
			return fmt.Errorf("load policy: %w", err)
		}
		loadedPolicy = &parsedPolicy
	}

	router := httpapi.NewRouterWithOptions(registryStore, httpapi.Options{
		AdminToken: config.AdminToken(root.adminToken),
		Policy:     loadedPolicy,
	})
	fmt.Fprintf(cmd.ErrOrStderr(), "listening on %s\n", addr)
	return router.Run(addr)
}
