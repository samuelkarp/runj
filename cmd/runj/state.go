package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// stateCommand implements the OCI "state" command
//
// state <container-id>
//
// This operation MUST generate an error if it is not provided the ID of a
// container. Attempting to query a container that does not exist MUST generate
// an error. This operation MUST return the state of a container as specified in
// the State section.
func stateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "state <container-id>",
		Short: "Query the state of a container",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(args)
		},
	}
}
