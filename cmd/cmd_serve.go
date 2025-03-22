package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/computer-technology-team/go-judge/cmd/serve"
	"github.com/computer-technology-team/go-judge/config"
)

func NewServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := cmd.Flags().GetString(configFileFlag)
			if err != nil {
				return fmt.Errorf("could not get config path flag: %w", err)
			}

			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("could not load config")
			}

			return serve.StartServer(cfg.Server)
		},
	}

	return cmd
}
