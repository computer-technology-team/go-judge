package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/computer-technology-team/go-judge/config"
	"github.com/computer-technology-team/go-judge/internal/storage"
)

func NewCreateAdminCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-admin",
		Short: "creates admin user, if user exists changes role to admin",
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

			_, err = storage.NewPgxPool(ctx, cfg.Database)
			if err != nil {
				return fmt.Errorf("could not create database pool: %w", err)
			}

			_ = storage.New()

			return nil
		},
	}

	return cmd
}
