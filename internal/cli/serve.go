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
	addAdminTokensFileFlag(command, root)
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

	adminTokens, adminTokensFile, err := loadAdminAuthConfig(root)
	if err != nil {
		return err
	}

	router := httpapi.NewRouterWithOptions(registryStore, httpapi.Options{
		AdminTokens:     adminTokens,
		AdminTokensFile: adminTokensFile,
		Policy:          loadedPolicy,
	})
	fmt.Fprintf(cmd.ErrOrStderr(), "listening on %s\n", addr)
	return router.Run(addr)
}

func loadAdminAuthConfig(root *rootOptions) ([]httpapi.AdminToken, string, error) {
	var tokens []httpapi.AdminToken
	if token := config.AdminToken(root.adminToken); token != "" {
		tokens = append(tokens, httpapi.AdminToken{
			Name:  "default-admin",
			Token: token,
			Role:  "admin",
		})
	}
	normalized, err := httpapi.NormalizeAdminTokens(tokens)
	if err != nil {
		return nil, "", fmt.Errorf("load admin tokens: %w", err)
	}
	tokensFile := config.AdminTokensFile(root.adminTokensFile)
	if tokensFile != "" {
		loadedTokens, err := httpapi.LoadAdminTokensFile(tokensFile)
		if err != nil {
			return nil, "", fmt.Errorf("load admin tokens: %w", err)
		}
		if _, err := httpapi.NormalizeAdminTokens(append(append([]httpapi.AdminToken{}, normalized...), loadedTokens...)); err != nil {
			return nil, "", fmt.Errorf("load admin tokens: %w", err)
		}
	}
	return normalized, tokensFile, nil
}

func loadAdminTokens(root *rootOptions) ([]httpapi.AdminToken, error) {
	staticTokens, tokensFile, err := loadAdminAuthConfig(root)
	if err != nil {
		return nil, err
	}
	tokens := append([]httpapi.AdminToken{}, staticTokens...)
	if tokensFile != "" {
		loadedTokens, err := httpapi.LoadAdminTokensFile(tokensFile)
		if err != nil {
			return nil, fmt.Errorf("load admin tokens: %w", err)
		}
		tokens = append(tokens, loadedTokens...)
	}
	normalized, err := httpapi.NormalizeAdminTokens(tokens)
	if err != nil {
		return nil, fmt.Errorf("load admin tokens: %w", err)
	}
	return normalized, nil
}
