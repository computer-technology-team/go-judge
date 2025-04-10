package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"

	"github.com/computer-technology-team/go-judge/config"
	"github.com/computer-technology-team/go-judge/internal/storage"
)

func NewCreateAdminCmd() *cobra.Command {
	var username, password string

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

			pool, err := storage.NewPgxPool(ctx, cfg.Database)
			if err != nil {
				return fmt.Errorf("could not create database pool: %w", err)
			}

			querier := storage.New()

			if username == "" || password == "" {
				return errors.New("username or password is empty")
			}

			passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				return fmt.Errorf("could not hash the password: %w", err)
			}

			_, err = querier.CreateAdmin(ctx, pool, username, string(passwordHash))
			if err != nil {
				return fmt.Errorf("could not create the admin in database: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&username, "username", "u", "", "the username of admin to create")
	cmd.Flags().StringVarP(&password, "password", "p", "", "the username of admin to create")

	return cmd
}
