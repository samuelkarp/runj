package containerd

import (
	"os"
	"os/exec"
	"testing"
)

func TestProcReapAcquire(t *testing.T) {
	const helperEnv = "RUNJ_TEST_PROC_REAP_ACQUIRE"

	if os.Getenv(helperEnv) == "1" {
		// Test the actual function in the child process.
		if err := procReapAcquire(); err != nil {
			t.Fatalf("procReapAcquire() failed: %v", err)
		}
		return
	}

	// Run this test again in a separate process.
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(executable, "-test.run=^TestProcReapAcquire$")
	cmd.Env = append(os.Environ(), helperEnv+"=1")

	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("subprocess failed: %v\n%s", err, output)
	}
}
