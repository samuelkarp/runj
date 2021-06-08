// +build integration

package integration

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"go.sbk.wtf/runj/runtimespec"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateDelete(t *testing.T) {
	dir, err := ioutil.TempDir("", "runj-integ-test-"+t.Name())
	require.NoError(t, err)

	tests := []runtimespec.Spec{
		// minimal
		{
			Process: &runtimespec.Process{},
		},
		// arguments
		{
			Process: &runtimespec.Process{
				Args: []string{"one", "two", "three"},
			},
		},
		// environment variables
		{
			Process: &runtimespec.Process{
				Env: []string{"one=two", "three=four", "five"},
			},
		},
	}

	for i, tc := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			bundleDir := filepath.Join(dir, strconv.Itoa(i))
			rootDir := filepath.Join(bundleDir, "root")
			err := os.MkdirAll(rootDir, 0755)
			require.NoError(t, err, "create bundle dir")
			t.Log("bundle", bundleDir)

			configJSON, err := json.Marshal(tc)
			require.NoError(t, err, "marshal config")
			err = ioutil.WriteFile(filepath.Join(bundleDir, "config.json"), configJSON, 0644)
			require.NoError(t, err, "write config")

			id := "test-create-delete-" + strconv.Itoa(i)
			cmd := exec.Command("runj", "create", id, bundleDir)
			cmd.Stdin = nil
			out, err := os.OpenFile(filepath.Join(bundleDir, "out"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
			require.NoError(t, err, "out file")
			cmd.Stdout = out
			cmd.Stderr = out
			err = cmd.Run()
			assert.NoError(t, err, "runj create")
			err = out.Close()
			assert.NoError(t, err, "out file close")
			outBytes, err := ioutil.ReadFile(filepath.Join(bundleDir, "out"))
			assert.NoError(t, err, "out file read")
			t.Log("runj create output:", string(outBytes))

			cmd = exec.Command("runj", "delete", id)
			cmd.Stdin = nil
			outBytes, err = cmd.CombinedOutput()
			assert.NoError(t, err, "runj delete")
			t.Log("runj delete output:", string(outBytes))
		})
	}
}
