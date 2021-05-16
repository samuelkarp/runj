package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"go.sbk.wtf/runj/jail"
	"go.sbk.wtf/runj/state"

	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
)

// killCommand implements the OCI "kill" command
//
// kill <container-id> <signal>
//
// This operation MUST generate an error if it is not provided the container ID.
// Attempting to send a signal to a container that is neither created nor
// running MUST have no effect on the container and MUST generate an error. This
// operation MUST send the specified signal to the container process.
//
// Extension: --pid argument is non-standard
func killCommand() *cobra.Command {
	kill := &cobra.Command{
		Use:   "kill <container-id> [signal]",
		Short: "Send a signal to a container",
		Long:  "Send a signal to a container.  If the signal is not specified, SIGTERM is sent.",
		Args:  cobra.RangeArgs(1, 2),
	}
	all := false
	kill.Flags().BoolVarP(
		&all,
		"all",
		"a",
		false,
		"send the specified signal to all processes inside the container")
	pid := 0
	kill.Flags().IntVarP(
		&pid,
		"pid",
		"p",
		0,
		"send the specified signal to a specific process ID")
	kill.PreRunE = func(cmd *cobra.Command, args []string) error {
		if all && pid != 0 {
			return errors.New("cannot specify both --all and --pid")
		}
		return nil
	}
	kill.RunE = func(cmd *cobra.Command, args []string) error {
		disableUsage(cmd)
		id := args[0]
		sigstr := "SIGTERM"
		if len(args) == 2 {
			sigstr = args[1]
		}
		signal, err := parseSignal(sigstr)
		if err != nil {
			return err
		}
		s, err := state.Load(id)
		if err != nil {
			return err
		}
		if s.Status == state.StatusRunning {
			if ok, err := jail.IsRunning(cmd.Context(), id, s.PID); err != nil {
				return err
			} else if !ok {
				s.Status = state.StatusStopped
				if err := s.Save(); err != nil {
					return err
				}
			}
		}
		if s.Status != state.StatusRunning {
			return errors.New("cannot signal non-running container")
		}
		if pid == 0 {
			pid = s.PID
		}
		if all {
			return jail.KillAll(cmd.Context(), id, signal)
		} else {
			return jail.Kill(cmd.Context(), id, pid, signal)
		}
	}
	return kill
}

// parseSignal was taken from runc
// Copyright 2012-2015 Docker, Inc.
// Licensed under the Apache License, 2.0
func parseSignal(rawSignal string) (unix.Signal, error) {
	s, err := strconv.Atoi(rawSignal)
	if err == nil {
		return unix.Signal(s), nil
	}
	sig := strings.ToUpper(rawSignal)
	if !strings.HasPrefix(sig, "SIG") {
		sig = "SIG" + sig
	}
	signal := unix.SignalNum(sig)
	if signal == 0 {
		return -1, fmt.Errorf("unknown signal %q", rawSignal)
	}
	return signal, nil
}

// End of runc code
