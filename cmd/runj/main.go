package main

import (
	"os"

	"github.com/spf13/cobra"
)

const (
	specConfig = "config.json"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "runj <command>",
		Short: "runj is a skeleton OCI runtime for FreeBSD",
	}
	rootCmd.AddCommand(stateCommand())
	rootCmd.AddCommand(createCommand())
	rootCmd.AddCommand(startCommand())
	rootCmd.AddCommand(killCommand())
	rootCmd.AddCommand(deleteCommand())
	rootCmd.AddCommand(demoCommand())
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func disableUsage(cmd *cobra.Command) {
	cmd.SetUsageFunc(func(*cobra.Command) error { return nil })
}
