package cmd

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/computer-technology-team/go-judge/config"
	authenticatorPkg "github.com/computer-technology-team/go-judge/internal/auth/authenticator"
)

func NewGenerateTokenCmd() *cobra.Command {
	var userID string
	cmd := &cobra.Command{
		Use:   "generate-token",
		Short: "Generate auth token for user id",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			configPath, err := cmd.Flags().GetString(configFileFlag)
			if err != nil {
				return fmt.Errorf("could not get config path flag: %w", err)
			}

			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("could not load config: %w", err)
			}

			authenticator, err := authenticatorPkg.NewAuthenticator(cfg.Authentication)
			if err != nil {
				return fmt.Errorf("could not create authenticator: %w", err)
			}

			if _, err := uuid.Parse(userID); err != nil {
				return fmt.Errorf("user id is not uuid: %w", err)
			}

			token, _, err := authenticator.GenerateToken(ctx, authenticatorPkg.Claims{UserID: userID})
			if err != nil {
				return fmt.Errorf("failed to generate token: %w", err)
			}

			fmt.Printf("Generated token successfully:\n%s\n", token)
			return nil
		},
	}

	cmd.Flags().StringVar(&userID, "user-id", "", "user id to generate token for.\n must be a valid uuid")

	return cmd
}
