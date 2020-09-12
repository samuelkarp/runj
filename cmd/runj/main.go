package main

import (
	"github.com/spf13/cobra"
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
	rootCmd.Execute()
}
