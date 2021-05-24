package main

import (
	"errors"
	"fmt"
	"os"

	"go.sbk.wtf/runj/jail"
	"go.sbk.wtf/runj/oci"
	"go.sbk.wtf/runj/runtimespec"
	"go.sbk.wtf/runj/state"

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
		RunE: func(cmd *cobra.Command, args []string) error {
			disableUsage(cmd)
			id := args[0]
			running, err := jail.IsRunning(cmd.Context(), id, 0)
			if err != nil {
				return fmt.Errorf("delete: failed to determine if jail is running: %w", err)
			}
			if running {
				return fmt.Errorf("delete: jail %s is not stopped", id)
			}
			err = jail.CleanupEntrypoint(id)
			if err != nil {
				return fmt.Errorf("delete: failed to find entrypoint process: %w", err)
			}
			confPath := jail.ConfPath(id)
			if _, err := os.Stat(confPath); err != nil {
				return errors.New("invalid jail id provided")
			}
			err = jail.DestroyJail(cmd.Context(), confPath, id)
			if err != nil {
				return err
			}
			var ociConfig *runtimespec.Spec
			ociConfig, err = oci.LoadConfig(id)
			if err != nil {
				return err
			}
			if ociConfig == nil {
				return errors.New("OCI config is required")
			}
			err = jail.Unmount(ociConfig)
			if err != nil {
				return err
			}
			return state.Remove(id)
		},
	}
}
