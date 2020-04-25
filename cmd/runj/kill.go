package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// killCommand implements the OCI "kill" command
//
// kill <container-id> <signal>
//
// This operation MUST generate an error if it is not provided the container ID.
// Attempting to send a signal to a container that is neither created nor
// running MUST have no effect on the container and MUST generate an error. This
// operation MUST send the specified signal to the container process.
func killCommand() *cobra.Command {
	kill := &cobra.Command{
		Use:   "kill <container-id> [signal]",
		Short: "Send a signal to a container",
		Long:  " Send a signal to a container.  If the signal is not specified, SIGTERM is sent.",
		Args:  cobra.RangeArgs(1, 2),
	}
	all := false
	kill.Flags().BoolVarP(
		&all,
		"all",
		"a",
		false,
		"send the specified signal to all processes inside the container")
	kill.Run = func(cmd *cobra.Command, args []string) {
		fmt.Println(args)
		fmt.Println(all)
	}
	return kill
}
