package cmd

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"google.golang.org/grpc/resolver"
)

const configFileFlag = "config"

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "go-judge",
		Short: "Run and Manage Go Judge",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			})
			slog.SetDefault(slog.New(logHandler))
		},
	}

	cmd.PersistentFlags().String(configFileFlag, "", "config file name")

	RegisterCommandRecursive(cmd)

	return cmd
}

func RegisterCommandRecursive(parent *cobra.Command) {
	serveCmd := NewServeCmd()

	runnerCmd := NewRunnerCmd()

	migrateCmd := NewMigrateCmd()
	generateTokenCmd := NewGenerateTokenCmd()

	createAdminCmd := NewCreateAdminCmd()

	parent.AddCommand(serveCmd, runnerCmd, migrateCmd, createAdminCmd, generateTokenCmd)
}

func Execute() {
	resolver.SetDefaultScheme("dns")

	err := NewRootCmd().Execute()
	if err != nil {
		slog.Error("error in executing command", "error", err)
		os.Exit(-1)
	}
}
