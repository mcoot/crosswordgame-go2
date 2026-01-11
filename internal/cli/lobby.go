package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newLobbyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lobby",
		Short: "Lobby management commands",
	}

	cmd.AddCommand(newLobbyCreateCmd())
	cmd.AddCommand(newLobbyGetCmd())
	cmd.AddCommand(newLobbyJoinCmd())
	cmd.AddCommand(newLobbyLeaveCmd())
	cmd.AddCommand(newLobbyConfigCmd())

	return cmd
}

func newLobbyCreateCmd() *cobra.Command {
	var gridSize int

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new lobby",
		RunE: func(cmd *cobra.Command, args []string) error {
			req := map[string]int{}
			if gridSize > 0 {
				req["grid_size"] = gridSize
			}

			var result Lobby

			if err := client.Post("/api/v1/lobbies", req, &result); err != nil {
				return err
			}

			out := NewOutput(cfg.Output)
			out.Print(result)
			return nil
		},
	}

	cmd.Flags().IntVar(&gridSize, "grid-size", 0, "Grid size (default: server default)")

	return cmd
}

func newLobbyGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <code>",
		Short: "Get lobby details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			code := args[0]

			var result Lobby

			if err := client.Get(fmt.Sprintf("/api/v1/lobbies/%s", code), &result); err != nil {
				return err
			}

			out := NewOutput(cfg.Output)
			out.Print(result)
			return nil
		},
	}
}

func newLobbyJoinCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "join <code>",
		Short: "Join a lobby",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			code := args[0]

			var result Lobby

			if err := client.Post(fmt.Sprintf("/api/v1/lobbies/%s/join", code), nil, &result); err != nil {
				return err
			}

			out := NewOutput(cfg.Output)
			out.Print(result)
			return nil
		},
	}
}

func newLobbyLeaveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "leave <code>",
		Short: "Leave a lobby",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			code := args[0]

			if err := client.Post(fmt.Sprintf("/api/v1/lobbies/%s/leave", code), nil, nil); err != nil {
				return err
			}

			out := NewOutput(cfg.Output)
			out.PrintMessage(fmt.Sprintf("Left lobby %s", code))
			return nil
		},
	}
}

func newLobbyConfigCmd() *cobra.Command {
	var gridSize int

	cmd := &cobra.Command{
		Use:   "config <code>",
		Short: "Update lobby configuration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			code := args[0]

			if gridSize == 0 {
				return fmt.Errorf("--grid-size is required")
			}

			req := map[string]int{"grid_size": gridSize}
			var result LobbyConfig

			if err := client.Patch(fmt.Sprintf("/api/v1/lobbies/%s/config", code), req, &result); err != nil {
				return err
			}

			out := NewOutput(cfg.Output)
			out.Print(result)
			return nil
		},
	}

	cmd.Flags().IntVar(&gridSize, "grid-size", 0, "Grid size (required)")
	_ = cmd.MarkFlagRequired("grid-size")

	return cmd
}
