package cli

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	cfg    *Config
	client *Client
)

// NewRootCmd creates the root command
func NewRootCmd() *cobra.Command {
	cfg = DefaultConfig()

	rootCmd := &cobra.Command{
		Use:   "cwgame",
		Short: "CLI tool for the crossword game API",
		Long: `cwgame is a CLI tool for interacting with the crossword game JSON API.

It supports all API operations including player management, lobby operations,
game actions, and real-time SSE event streaming.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Load token from file if not provided via flag/env
			if err := cfg.LoadToken(); err != nil {
				return err
			}

			// Create HTTP client
			client = NewClient(cfg.ServerURL, cfg.Token)
			return nil
		},
		SilenceUsage: true,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfg.ServerURL, "server", cfg.ServerURL, "Server URL (env: CWGAME_SERVER)")
	rootCmd.PersistentFlags().StringVar(&cfg.Token, "token", cfg.Token, "Session token (env: CWGAME_TOKEN)")
	rootCmd.PersistentFlags().StringVar(&cfg.TokenFile, "token-file", cfg.TokenFile, "Token file path (env: CWGAME_TOKEN_FILE)")
	rootCmd.PersistentFlags().StringVarP(&cfg.Output, "output", "o", cfg.Output, "Output format: text, json")
	rootCmd.PersistentFlags().BoolVarP(&cfg.Verbose, "verbose", "v", cfg.Verbose, "Verbose output")

	// Add subcommands
	rootCmd.AddCommand(newPlayerCmd())
	rootCmd.AddCommand(newLobbyCmd())
	rootCmd.AddCommand(newGameCmd())
	rootCmd.AddCommand(newEventsCmd())
	rootCmd.AddCommand(newHealthCmd())

	return rootCmd
}

// Execute runs the root command
func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
