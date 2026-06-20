package cli

import (
	"github.com/spf13/cobra"
)

type rootOptions struct {
	databaseURL string
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
		Use:   "ard-server",
		Short: "Run the ARD registry server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServer(cmd, &options, addr)
		},
	}
	addDatabaseFlag(command, &options)
	command.Flags().StringVar(&addr, "addr", ":8080", "HTTP listen address")
	return command
}

func newRootCommand(use string, includeServer bool) *cobra.Command {
	options := rootOptions{}
	command := &cobra.Command{
		Use:   use,
		Short: "Self-hosted Agentic Resource Discovery registry and toolkit",
	}
	addDatabaseFlag(command, &options)

	if includeServer {
		command.AddCommand(newServeCommand(&options))
	}
	command.AddCommand(newAddCommand(&options))
	command.AddCommand(newCrawlCommand(&options))
	command.AddCommand(newSearchCommand())
	command.AddCommand(newVerifyCommand())
	return command
}

func addDatabaseFlag(command *cobra.Command, options *rootOptions) {
	command.PersistentFlags().StringVar(
		&options.databaseURL,
		"database-url",
		"",
		"Postgres connection URL. Defaults to DATABASE_URL or local postgres.",
	)
}
