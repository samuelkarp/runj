package main

import (
	"encoding/json"
	"fmt"

	"go.sbk.wtf/runj/jail"
	"go.sbk.wtf/runj/runtimespec"
	"go.sbk.wtf/runj/state"

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
//
// The state of a container includes the following properties:
//
// * ociVersion (string, REQUIRED) is version of the Open Container Initiative
//   Runtime Specification with which the state complies.
// * id (string, REQUIRED) is the container's ID. This MUST be unique across all
//   containers on this host. There is no requirement that it be unique across
//   hosts.
// * status (string, REQUIRED) is the runtime state of the container. The value
//   MAY be one of:
//   * creating: the container is being created (step 2 in the lifecycle)
//   * created: the runtime has finished the create operation (after step 2 in
//     the lifecycle), and the container process has neither exited nor executed
//     the user-specified program
//   * running: the container process has executed the user-specified program
//     but has not exited (after step 5 in the lifecycle)
//   * stopped: the container process has exited (step 7 in the lifecycle)
//   Additional values MAY be defined by the runtime, however, they MUST be used
//   to represent new runtime states not defined above.
// * pid (int, REQUIRED when status is created or running on Linux, OPTIONAL on
//   other platforms) is the ID of the container process. For hooks executed in
//   the runtime namespace, it is the pid as seen by the runtime. For hooks
//   executed in the container namespace, it is the pid as seen by the
//   container.
// * bundle (string, REQUIRED) is the absolute path to the container's bundle
//   directory. This is provided so that consumers can find the container's
//   configuration and root filesystem on the host.
// * annotations (map, OPTIONAL) contains the list of annotations associated
//   with the container. If no annotations were provided then this property MAY
//   either be absent or an empty map.
// The state MAY include additional properties.
func stateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "state <container-id>",
		Short: "Query the state of a container",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			disableUsage(cmd)
			id := args[0]
			s, err := state.Load(id)
			if err != nil {
				return err
			}
			if s.Status == state.StatusRunning {
				ok, err := jail.IsRunning(cmd.Context(), id, s.PID)
				if err != nil {
					return err
				}
				if !ok {
					s.Status = state.StatusStopped
					s.PID = 0
					err = s.Save()
					if err != nil {
						return err
					}
				}
			}
			output := StateOutput{
				OCIVersion: runtimespec.Version,
				ID:         id,
				Status:     string(s.Status),
				PID:        s.PID,
				Bundle:     s.Bundle,
			}
			b, err := json.MarshalIndent(output, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(b))
			return nil
		},
	}
}

// StateOutput is the expected output format for the state command
/*
{
    "ociVersion": "0.2.0",
    "id": "oci-container1",
    "status": "running",
    "pid": 4422,
    "bundle": "/containers/redis",
    "annotations": {
        "myKey": "myValue"
    }
}
*/
type StateOutput struct {
	OCIVersion  string            `json:"ociVersion"`
	ID          string            `json:"id"`
	Status      string            `json:"status"`
	PID         int               `json:"pid,omitempty"`
	Bundle      string            `json:"bundle"`
	Annotations map[string]string `json:"annotations,omitempty"`
}
