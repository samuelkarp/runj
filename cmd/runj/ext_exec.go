package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"
	"go.sbk.wtf/runj/jail"
	"go.sbk.wtf/runj/oci"
	"go.sbk.wtf/runj/runtimespec"
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
	processJSONFlag := execCmd.Flags().StringP("process", "p", "", "process.json")
	consoleSocket := execCmd.Flags().String(
		"console-socket",
		"",
		`path to an AF_UNIX socket which will receive a
file descriptor referencing the master end of
the console's pseudoterminal`)
	execCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if processJSONFlag == nil || *processJSONFlag == "" {
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
		if processJSONFlag != nil && *processJSONFlag != "" {
			data, err := ioutil.ReadFile(*processJSONFlag)
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
		// console socket validation
		if process.Terminal {
			if *consoleSocket == "" {
				return errors.New("console-socket is required when Process.Terminal is true")
			}
			if socketStat, err := os.Stat(*consoleSocket); err != nil {
				return fmt.Errorf("failed to stat console socket %q: %w", *consoleSocket, err)
			} else if socketStat.Mode()&os.ModeSocket != os.ModeSocket {
				return fmt.Errorf("console-socket %q is not a socket", *consoleSocket)
			}
		} else if *consoleSocket != "" {
			return errors.New("console-socket provided but Process.Terminal is false")
		}

		cmd.SilenceErrors = true
		// Setup and start the "runj-entrypoint" helper program in order to
		// get the container STDIO hooked up properly.
		return jail.ExecEntrypoint(id, process.Args, process.Env, *consoleSocket)
	}
	return execCmd
}
