package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "runj <command>",
		Short:   "runj is a skeleton OCI runtime for FreeBSD",
		Version: version(),
	}
	rootCmd.AddCommand(stateCommand())
	rootCmd.AddCommand(createCommand())
	rootCmd.AddCommand(startCommand())
	rootCmd.AddCommand(killCommand())
	rootCmd.AddCommand(deleteCommand())
	rootCmd.AddCommand(extCommand())
	rootCmd.AddCommand(demoCommand())
	err := rootCmd.Execute()
	if err != nil {
		code := 1
		if e, ok := err.(*exec.ExitError); ok {
			code = e.ExitCode()
		}
		os.Exit(code)
	}
}

func version() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	sb := strings.Builder{}
	sb.WriteString(bi.Main.Version + "\n")
	sb.WriteString("go: " + bi.GoVersion)
	for _, dep := range bi.Deps {
		sb.WriteString(fmt.Sprintf("\n%s: %s", dep.Path, dep.Version))
	}
	return sb.String()
}

// disableUsage is a helper to disable the Usage output on errors.  This helper
// is used because we want usage output for input validation errors (wrong
// number of arguments, wrong type, etc) in both the cobra-provided validations
// and in PreRunE funcs, but we don't want that output for the actual command
// execution (RunE funcs).
func disableUsage(cmd *cobra.Command) {
	cmd.SetUsageFunc(func(*cobra.Command) error { return nil })
}
