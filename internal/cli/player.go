package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newPlayerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "player",
		Short: "Player management commands",
	}

	cmd.AddCommand(newPlayerGuestCmd())
	cmd.AddCommand(newPlayerRegisterCmd())
	cmd.AddCommand(newPlayerLoginCmd())
	cmd.AddCommand(newPlayerMeCmd())

	return cmd
}

func newPlayerGuestCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "guest",
		Short: "Create a guest player",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}

			req := map[string]string{"display_name": name}
			var result AuthResult

			if err := client.Post("/api/v1/players/guest", req, &result); err != nil {
				return err
			}

			// Save token
			if err := cfg.SaveToken(result.SessionToken); err != nil {
				return fmt.Errorf("failed to save token: %w", err)
			}

			out := NewOutput(cfg.Output)
			out.Print(result)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Display name (required)")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}

func newPlayerRegisterCmd() *cobra.Command {
	var name, user, pass string

	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register a new player account",
		RunE: func(cmd *cobra.Command, args []string) error {
			if user == "" || pass == "" || name == "" {
				return fmt.Errorf("--name, --user, and --pass are required")
			}

			req := map[string]string{
				"display_name": name,
				"username":     user,
				"password":     pass,
			}
			var result AuthResult

			if err := client.Post("/api/v1/players/register", req, &result); err != nil {
				return err
			}

			// Save token
			if err := cfg.SaveToken(result.SessionToken); err != nil {
				return fmt.Errorf("failed to save token: %w", err)
			}

			out := NewOutput(cfg.Output)
			out.Print(result)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Display name (required)")
	cmd.Flags().StringVar(&user, "user", "", "Username (required)")
	cmd.Flags().StringVar(&pass, "pass", "", "Password (required)")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("user")
	_ = cmd.MarkFlagRequired("pass")

	return cmd
}

func newPlayerLoginCmd() *cobra.Command {
	var user, pass string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login with existing account",
		RunE: func(cmd *cobra.Command, args []string) error {
			if user == "" || pass == "" {
				return fmt.Errorf("--user and --pass are required")
			}

			req := map[string]string{
				"username": user,
				"password": pass,
			}
			var result AuthResult

			if err := client.Post("/api/v1/players/login", req, &result); err != nil {
				return err
			}

			// Save token
			if err := cfg.SaveToken(result.SessionToken); err != nil {
				return fmt.Errorf("failed to save token: %w", err)
			}

			out := NewOutput(cfg.Output)
			out.Print(result)
			return nil
		},
	}

	cmd.Flags().StringVar(&user, "user", "", "Username (required)")
	cmd.Flags().StringVar(&pass, "pass", "", "Password (required)")
	_ = cmd.MarkFlagRequired("user")
	_ = cmd.MarkFlagRequired("pass")

	return cmd
}

func newPlayerMeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "me",
		Short: "Show current player info",
		RunE: func(cmd *cobra.Command, args []string) error {
			var result Player

			if err := client.Get("/api/v1/players/me", &result); err != nil {
				return err
			}

			out := NewOutput(cfg.Output)
			out.Print(result)
			return nil
		},
	}
}
