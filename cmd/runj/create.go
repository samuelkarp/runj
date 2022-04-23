package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"go.sbk.wtf/runj/jail"
	"go.sbk.wtf/runj/oci"
	"go.sbk.wtf/runj/runtimespec"
	"go.sbk.wtf/runj/state"

	"github.com/spf13/cobra"
)

// createCommand implements the OCI "create" command
//
// create <container-id> <path-to-bundle>
//
// This operation MUST generate an error if it is not provided a path to the
// bundle and the container ID to associate with the container. If the ID
// provided is not unique across all containers within the scope of the runtime,
// or is not valid in any other way, the implementation MUST generate an error
// and a new container MUST NOT be created. This operation MUST create
// a new container.
//
// All of the properties configured in config.json except for process MUST be
// applied. process.args MUST NOT be applied until triggered by the start
// operation. The remaining process properties MAY be applied by this operation.
// If the runtime cannot apply a property as specified in the configuration, it
// MUST generate an error and a new container MUST NOT be created.
//
// The runtime MAY validate config.json against this spec, either generically or
// with respect to the local system capabilities, before creating the container
// (step 2). Runtime callers who are interested in pre-create validation can run
// bundle-validation tools before invoking the create operation.
//
// Any changes made to the config.json file after this operation will not have
// an effect on the container.
//
// runc's implementation of `create` hooks up the container process's STDIO to
// the same STDIO streams used for the invocation  of `runc create`.  Because
// integrations on top of runc expect this behavior, runj copies that at the
// expense of more complication in the codebase.
func createCommand() *cobra.Command {
	var (
		bundle        string
		consoleSocket string
		pidFile       string
	)

	create := &cobra.Command{
		Use:   "create <container-id> [<path-to-bundle]",
		Short: "Create a new container with given ID and bundle",
		Long: `Create a new container with given ID and bundle.  IDs must be unique.

The create command creates an instance of a container for a bundle. The bundle
is a directory with a specification file named "config.json" and a root
filesystem.

The specification file includes an args parameter. The args parameter is used
to specify command(s) that get run when the container is started. To change the
command(s) that get executed on start, edit the args parameter of the spec.`,
		Args: cobra.RangeArgs(1, 2),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 2 {
				bundle = args[1]
			}
			bundleConfig := filepath.Join(bundle, oci.ConfigFileName)
			fInfo, err := os.Stat(bundleConfig)
			if err != nil {
				return err
			}
			if fInfo.Mode()&os.ModeType != 0 {
				return fmt.Errorf("%q should be a regular file", bundleConfig)
			}
			return nil
		},
	}
	flags := create.Flags()
	flags.StringVar(
		&bundle,
		"bundle",
		"",
		`path to the root of the bundle directory, defaults to the current directory`)
	flags.StringVar(
		&consoleSocket,
		"console-socket",
		"",
		`path to an AF_UNIX socket which will receive a
file descriptor referencing the master end of
the console's pseudoterminal`)
	flags.StringVar(
		&pidFile,
		"pid-file",
		"",
		"the file to write the process id to")
	create.RunE = func(cmd *cobra.Command, args []string) (err error) {
		disableUsage(cmd)
		id := args[0]
		var s *state.State
		s, err = state.Create(id, bundle)
		if err != nil {
			return err
		}
		defer func() {
			if err == nil {
				s.Status = state.StatusCreated
				err = s.Save()
			} else {
				state.Remove(id)
			}
		}()
		err = oci.StoreConfig(id, bundle)
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
		if ociConfig.Process == nil {
			return errors.New("OCI config Process is required")
		}
		rootPath := filepath.Join(bundle, "root")
		if ociConfig.Root != nil && ociConfig.Root.Path != "" {
			rootPath = ociConfig.Root.Path
			if rootPath[0] != filepath.Separator {
				rootPath = filepath.Join(bundle, rootPath)
			}
			ociConfig.Root.Path = rootPath
		} else {
			ociConfig.Root = &runtimespec.Root{Path: rootPath}
		}
		// console socket validation
		if ociConfig.Process.Terminal {
			if consoleSocket == "" {
				return errors.New("console-socket is required when Process.Terminal is true")
			}
			if socketStat, err := os.Stat(consoleSocket); err != nil {
				return fmt.Errorf("failed to stat console socket %q: %w", consoleSocket, err)
			} else if socketStat.Mode()&os.ModeSocket != os.ModeSocket {
				return fmt.Errorf("console-socket %q is not a socket", consoleSocket)
			}
		} else if consoleSocket != "" {
			return errors.New("console-socket provided but Process.Terminal is false")
		}
		var confPath string
		confPath, err = jail.CreateConfig(id, rootPath)
		if err != nil {
			return err
		}
		if err := jail.CreateJail(cmd.Context(), confPath); err != nil {
			return err
		}
		err = jail.Mount(ociConfig)
		if err != nil {
			return err
		}
		defer func() {
			if err == nil {
				return
			}
			jail.Unmount(ociConfig)
		}()

		// Setup and start the "runj-entrypoint" helper program in order to
		// get the container STDIO hooked up properly.
		var entrypoint *exec.Cmd
		entrypoint, err = jail.SetupEntrypoint(id, true, ociConfig.Process.Args, ociConfig.Process.Env, consoleSocket)
		if err != nil {
			return err
		}
		// the runj-entrypoint pid will become the container process's pid
		// through a series of exec(2) calls
		s.PID = entrypoint.Process.Pid
		if pidFile != "" {
			f, err := os.OpenFile(pidFile, os.O_RDWR|os.O_CREATE, 0o666)
			if err != nil {
				return err
			}
			pidValue := strconv.Itoa(s.PID)
			_, err = f.WriteString(pidValue)
			f.Close()
			if err != nil {
				return err
			}
		}

		return nil
	}
	return create
}
