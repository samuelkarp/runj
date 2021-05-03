package main

import "github.com/spf13/cobra"

// demoCommand provides a subcommand for runj-specific demos.
// This command and its subcommands are not part of the OCI spec.
func extCommand() *cobra.Command {
	ext := &cobra.Command{
		Use:     "extension",
		Aliases: []string{"ext"},
		Short:   "Extensions for the OCI spec",
	}
	ext.AddCommand(execCommand())
	return ext
}
