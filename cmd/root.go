package cmd

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

const configFileFlag = "config"

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "go-judge",
		Short: "Run and Manage Go Judge",
	}

	cmd.PersistentFlags().String(configFileFlag, "", "config file name")

	RegisterCommandRecursive(cmd)

	return cmd
}

func RegisterCommandRecursive(parent *cobra.Command) {
	serveCmd := NewServeCmd()

	migrateCmd := NewMigrateCmd()

	parent.AddCommand(serveCmd, migrateCmd)
}

func Execute() {
	err := NewRootCmd().Execute()
	if err != nil {
		slog.Error("error in executing command", "error", err)
		os.Exit(-1)
	}
}
