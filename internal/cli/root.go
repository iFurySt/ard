package cli

import (
	"fmt"

	"github.com/ifuryst/ard/internal/buildinfo"
	"github.com/spf13/cobra"
)

type rootOptions struct {
	databaseURL     string
	adminToken      string
	adminTokensFile string
	policyFile      string
}

func NewRootCommand() *cobra.Command {
	return newRootCommand("ard", true)
}

func NewCLICommand() *cobra.Command {
	return newRootCommand("ardctl", false)
}

func NewServerCommand() *cobra.Command {
	options := rootOptions{}
	var addr string
	command := &cobra.Command{
		Use:     "ard-server",
		Short:   "Run the ARD registry server",
		Version: buildinfo.Version,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServer(cmd, &options, addr)
		},
	}
	command.SetVersionTemplate(versionTemplate(command.Use))
	addDatabaseFlag(command, &options)
	addAdminTokenFlag(command, &options)
	addAdminTokensFileFlag(command, &options)
	addPolicyFlag(command, &options)
	command.Flags().StringVar(&addr, "addr", ":8080", "HTTP listen address")
	command.AddCommand(newVersionCommand())
	return command
}

func newRootCommand(use string, includeServer bool) *cobra.Command {
	options := rootOptions{}
	command := &cobra.Command{
		Use:     use,
		Short:   "Self-hosted Agentic Resource Discovery registry and toolkit",
		Version: buildinfo.Version,
	}
	command.SetVersionTemplate(versionTemplate(use))
	addDatabaseFlag(command, &options)
	addPolicyFlag(command, &options)

	if includeServer {
		command.AddCommand(newServeCommand(&options))
	}
	command.AddCommand(newAddCommand(&options))
	command.AddCommand(newAdminCommand())
	command.AddCommand(newBrowseCommand())
	command.AddCommand(newCrawlCommand(&options))
	command.AddCommand(newExportCommand(&options))
	command.AddCommand(newHealthCommand())
	command.AddCommand(newListCommand(&options))
	command.AddCommand(newMetricsCommand())
	command.AddCommand(newRemoveCommand(&options))
	command.AddCommand(newSearchCommand())
	command.AddCommand(newVerifyCommand(&options))
	command.AddCommand(newVersionCommand())
	return command
}

func versionTemplate(name string) string {
	return fmt.Sprintf("%s %s\n", name, buildinfo.Current().String())
}

func addDatabaseFlag(command *cobra.Command, options *rootOptions) {
	command.PersistentFlags().StringVar(
		&options.databaseURL,
		"database-url",
		"",
		"Postgres connection URL. Defaults to DATABASE_URL or local postgres.",
	)
}

func addAdminTokenFlag(command *cobra.Command, options *rootOptions) {
	command.Flags().StringVar(
		&options.adminToken,
		"admin-token",
		"",
		"Bearer token for protected admin API routes. Defaults to ARD_ADMIN_TOKEN.",
	)
}

func addAdminTokensFileFlag(command *cobra.Command, options *rootOptions) {
	command.Flags().StringVar(
		&options.adminTokensFile,
		"admin-tokens-file",
		"",
		"Optional admin token role file. Defaults to ARD_ADMIN_TOKENS_FILE.",
	)
}

func addPolicyFlag(command *cobra.Command, options *rootOptions) {
	command.PersistentFlags().StringVar(
		&options.policyFile,
		"policy-file",
		"",
		"Optional ingestion policy JSON file. Defaults to ARD_POLICY_FILE.",
	)
}
