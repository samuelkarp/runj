package jail

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// CreateJail wraps the jail(8) command to create a jail
func CreateJail(ctx context.Context, confPath string) error {
	cmd := exec.CommandContext(ctx, "jail", "-cf", confPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintln(os.Stderr, string(out))
	}
	return err
}

// DestroyJail wraps the jail(8) command to destroy a jail
func DestroyJail(ctx context.Context, confPath, jail string) error {
	cmd := exec.CommandContext(ctx, "jail", "-f", confPath, "-r", jail)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintln(os.Stderr, string(out))
	}
	return err
}
