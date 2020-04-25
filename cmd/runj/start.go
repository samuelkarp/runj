package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// startCommand implements the OCI "start" command
//
// start <container-id>
//
// This operation MUST generate an error if it is not provided the container ID.
// Attempting to start a container that is not created MUST have no effect on
// the container and MUST generate an error. This operation MUST run the
// user-specified program as specified by process. This operation MUST generate
// an error if process was not set.
func startCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "start <container-id>",
		Short: "Start a container",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(args)
		},
	}
}
