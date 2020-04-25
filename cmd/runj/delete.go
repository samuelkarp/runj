package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// deleteContainer implements the OCI "delete" command
//
// delete <container-id>
//
// This operation MUST generate an error if it is not provided the container ID.
// Attempting to delete a container that is not stopped MUST have no effect on
// the container and MUST generate an error. Deleting a container MUST delete
// the resources that were created during the create step. Note that resources
// associated with the container, but not created by this container, MUST NOT be
// deleted. Once a container is deleted its ID MAY be used by a subsequent
// container.
func deleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <container-id>",
		Short: "Delete a container",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(args)
		},
	}
}
