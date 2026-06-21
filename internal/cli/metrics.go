package cli

import (
	"context"
	"fmt"

	"github.com/ifuryst/ard/pkg/client"
	"github.com/spf13/cobra"
)

func newMetricsCommand() *cobra.Command {
	var registryURL string
	command := &cobra.Command{
		Use:   "metrics",
		Short: "Fetch public ARD registry metrics",
		RunE: func(cmd *cobra.Command, args []string) error {
			registry, err := client.New(registryURL, client.WithUserAgent("ardctl/0.1"))
			if err != nil {
				return err
			}
			metrics, err := registry.Metrics(context.Background())
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), metrics)
			if metrics == "" || metrics[len(metrics)-1] != '\n' {
				fmt.Fprintln(cmd.OutOrStdout())
			}
			return nil
		},
	}
	command.Flags().StringVar(&registryURL, "registry-url", envOrDefault("ARD_REGISTRY_URL", "http://127.0.0.1:8080"), "ARD registry base URL")
	return command
}
