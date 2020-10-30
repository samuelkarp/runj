package oci

import (
	"io"
	"os"
	"path/filepath"

	"go.sbk.wtf/runj/state"
)

const (
	ConfigFileName = "config.json"
)

func StoreConfig(id, bundlePath string) error {
	input, err := os.OpenFile(filepath.Join(bundlePath, ConfigFileName), os.O_RDONLY, 0)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(filepath.Join(state.Dir(id), ConfigFileName), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer func() {
		output.Close()
		if err != nil {
			os.Remove(output.Name())
		}
	}()
	_, err = io.Copy(output, input)
	return err
}
