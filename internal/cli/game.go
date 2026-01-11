package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func newGameCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "game",
		Short: "Game commands",
	}

	cmd.AddCommand(newGameStartCmd())
	cmd.AddCommand(newGameGetCmd())
	cmd.AddCommand(newGameAnnounceCmd())
	cmd.AddCommand(newGamePlaceCmd())
	cmd.AddCommand(newGameAbandonCmd())

	return cmd
}

func newGameStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start <code>",
		Short: "Start a new game in the lobby",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			code := args[0]

			var result GameState

			if err := client.Post(fmt.Sprintf("/api/v1/lobbies/%s/game", code), nil, &result); err != nil {
				return err
			}

			out := NewOutput(cfg.Output)
			out.Print(result)
			return nil
		},
	}
}

func newGameGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <code>",
		Short: "Get current game state",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			code := args[0]

			var result GameState

			if err := client.Get(fmt.Sprintf("/api/v1/lobbies/%s/game", code), &result); err != nil {
				return err
			}

			out := NewOutput(cfg.Output)
			out.Print(result)
			return nil
		},
	}
}

func newGameAnnounceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "announce <code> <letter>",
		Short: "Announce a letter (announcer only)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			code := args[0]
			letter := strings.ToUpper(args[1])

			if len(letter) != 1 || letter[0] < 'A' || letter[0] > 'Z' {
				return fmt.Errorf("letter must be a single character A-Z")
			}

			req := map[string]string{"letter": letter}
			var result AnnounceResult

			if err := client.Post(fmt.Sprintf("/api/v1/lobbies/%s/game/announce", code), req, &result); err != nil {
				return err
			}

			out := NewOutput(cfg.Output)
			out.Print(result)
			return nil
		},
	}
}

func newGamePlaceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "place <code> <row> <col>",
		Short: "Place the current letter on your board",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			code := args[0]

			row, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid row: %w", err)
			}

			col, err := strconv.Atoi(args[2])
			if err != nil {
				return fmt.Errorf("invalid col: %w", err)
			}

			req := map[string]int{"row": row, "col": col}
			var result PlaceResult

			if err := client.Post(fmt.Sprintf("/api/v1/lobbies/%s/game/place", code), req, &result); err != nil {
				return err
			}

			out := NewOutput(cfg.Output)
			out.Print(result)
			return nil
		},
	}
}

func newGameAbandonCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "abandon <code>",
		Short: "Abandon the current game (host only)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			code := args[0]

			if err := client.Delete(fmt.Sprintf("/api/v1/lobbies/%s/game", code)); err != nil {
				return err
			}

			out := NewOutput(cfg.Output)
			out.PrintMessage("Game abandoned")
			return nil
		},
	}
}
