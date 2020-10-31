package jail

import (
	"os"
	"os/exec"
)

// ExecAsync runs the specified command in the specified jail, without waiting
// for the process to complete.
// Note: this API is unstable; expect it to change.
func ExecAsync(id string, argv []string) (int, error) {
	args := append([]string{id}, argv...)
	cmd := exec.Command("jexec", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		return -1, err
	}
	return cmd.Process.Pid, nil
}
