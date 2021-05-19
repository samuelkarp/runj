package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os/exec"

	"go.sbk.wtf/runj/oci"
	"go.sbk.wtf/runj/runtimespec"

	"github.com/spf13/cobra"
	"go.sbk.wtf/runj/jail"
	"go.sbk.wtf/runj/state"
)

// execCommand implements the "exec" command, which is not part of the OCI spec
// and is instead patterned against the "exec" command from runc.
//
// exec <container-id> <command>
// or
// exec -p <process.json> <container-id>
//
// This operation combines parts of the "create" and "start" command, but is
// different from both.  Like "create", "exec" is responsible for configuring
// process STDIO and the environment.  Like "start", the process is started as
// a result of running "exec".  Unlike "create", the process starts immediately.
// Unlike "start", runj does not exit and instead exec's into the process.
func execCommand() *cobra.Command {
	execCmd := &cobra.Command{
		Use:   "exec <container-id> [-p <process.json>] [<command>]",
		Short: "exec a new process in a jail",
		Long:  "The exec command executes a new process in the context of an existing jail",
		Args:  cobra.MinimumNArgs(1),
	}
	processJsonFlag := execCmd.Flags().StringP("process", "p", "", "process.json")
	execCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if processJsonFlag == nil || *processJsonFlag == "" {
			// 2 args are required when -p not specified
			return cobra.MinimumNArgs(2)(cmd, args)
		}
		return cobra.ExactArgs(1)(cmd, args)
	}
	execCmd.RunE = func(cmd *cobra.Command, args []string) error {
		disableUsage(cmd)
		id := args[0]

		s, err := state.Load(id)
		if err != nil {
			return err
		}
		if s.Status != state.StatusRunning {
			return errors.New("cannot exec non-running container")
		}
		if ok, err := jail.IsRunning(cmd.Context(), id, s.PID); !ok {
			return errors.New("cannot exec non-running container")
		} else if err != nil {
			return err
		}

		var process runtimespec.Process
		if processJsonFlag != nil && *processJsonFlag != "" {
			data, err := ioutil.ReadFile(*processJsonFlag)
			if err != nil {
				return err
			}
			err = json.Unmarshal(data, &process)
			if err != nil {
				return err
			}
		} else {
			// populate process from the bundle
			ociConfig, err := oci.LoadConfig(id)
			if err != nil {
				return err
			}
			process = *ociConfig.Process
			process.Args = args[1:]
		}

		cmd.SilenceErrors = true
		// Setup and start the "runj-entrypoint" helper program in order to
		// get the container STDIO hooked up properly.
		var entrypoint *exec.Cmd
		entrypoint, err = jail.SetupEntrypoint(id, false, process.Args, process.Env, "")
		if err != nil {
			return err
		}
		return entrypoint.Wait()
	}
	return execCmd
}
