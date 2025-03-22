package cmd

import (
	"fmt"

	"github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/spf13/cobra"

	"github.com/computer-technology-team/go-judge/config"
	"github.com/computer-technology-team/go-judge/internal/storage"
)

func NewMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate the database",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := cmd.Flags().GetString(configFileFlag)
			if err != nil {
				return fmt.Errorf("could not get config path flag: %w", err)
			}

			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("could not load config: %w", err)
			}

			postgresDriver := pgx.Postgres{}
			dbDriver, err := postgresDriver.Open(cfg.Database.DSN())
			if err != nil {
				return fmt.Errorf("could not create migration connection: %w", err)
			}

			return storage.EnsureMigrationsDone(dbDriver, cfg.Database.Name)
		},
	}

	return cmd
}
