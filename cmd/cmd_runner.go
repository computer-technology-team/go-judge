package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/computer-technology-team/go-judge/cmd/runner"
	"github.com/computer-technology-team/go-judge/config"
)

func NewRunnerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "runner",
		Short: "Start the Runner service",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := cmd.Flags().GetString(configFileFlag)
			if err != nil {
				return fmt.Errorf("could not get config path flag: %w", err)
			}

			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("could not load config: %w", err)
			}

			return runner.StartServer(cmd.Context(), *cfg)
		},
	}

	return cmd
}
