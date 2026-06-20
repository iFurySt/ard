package cli

import (
	"github.com/spf13/cobra"
)

type rootOptions struct {
	databaseURL string
}

func NewRootCommand() *cobra.Command {
	options := rootOptions{}
	command := &cobra.Command{
		Use:   "ard",
		Short: "Self-hosted Agentic Resource Discovery registry and toolkit",
	}
	command.PersistentFlags().StringVar(
		&options.databaseURL,
		"database-url",
		"",
		"Postgres connection URL. Defaults to DATABASE_URL or local postgres.",
	)

	command.AddCommand(newServeCommand(&options))
	command.AddCommand(newAddCommand(&options))
	command.AddCommand(newSearchCommand())
	return command
}
