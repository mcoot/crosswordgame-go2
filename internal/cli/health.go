package cli

import (
	"github.com/spf13/cobra"
)

func newHealthCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Check server health",
		RunE: func(cmd *cobra.Command, args []string) error {
			var result HealthResult

			if err := client.Get("/api/v1/health", &result); err != nil {
				return err
			}

			out := NewOutput(cfg.Output)
			out.Print(result)
			return nil
		},
	}
}
