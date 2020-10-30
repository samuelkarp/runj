package jail

import (
	"os"
	"os/exec"
)

// ExecAsync runs the specified command in the specified jail, without waiting
// for the process to complete.
// Note: this API is unstable; expect it to change.
func ExecAsync(id string, argv []string) error {
	args := append([]string{id}, argv...)
	cmd := exec.Command("jexec", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Start()
}
