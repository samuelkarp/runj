package main

import (
	"errors"

	"go.sbk.wtf/runj/jail"
	"go.sbk.wtf/runj/oci"
	"go.sbk.wtf/runj/state"

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
//
// runc's implementation of the start command exits immediately after starting
// the container's process.  This does not appear to be specified in the spec.
func startCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "start <container-id>",
		Short: "Start a jail",
		Long:  "The start command executes the user-defined process in a created jail",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			disableUsage(cmd)
			id := args[0]
			ociConfig, err := oci.LoadConfig(id)
			if err != nil {
				return err
			}
			if ociConfig == nil || ociConfig.Process == nil || len(ociConfig.Process.Args) == 0 {
				return errors.New("start: missing process")
			}
			s, err := state.Load(id)
			if err != nil {
				return err
			}
			if s.Status == state.StatusRunning {
				if ok, err := jail.IsRunning(cmd.Context(), id); ok {
					return errors.New("cannot start already running container")
				} else if err != nil {
					return err
				}
			}
			err = jail.AwaitFifoOpen(cmd.Context(), id)
			if err != nil {
				return err
			}
			s.Status = state.StatusRunning
			return s.Save()
		},
	}
}
