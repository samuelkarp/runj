package oci

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"go.sbk.wtf/runj/runtimespec"

	"go.sbk.wtf/runj/state"
)

const (
	// ConfigFileName is the name of the config file
	ConfigFileName = "config.json"
)

// StoreConfig copies the config file provided in the input bundle to the state
// directory for the container.  The file must be copied to comply with this
// requirement from the OCI runtime specification:
// Any changes made to the config.json file after this operation will not have
// an effect on the container.
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

// LoadConfig loads the config file stored in the state directory
func LoadConfig(id string) (*runtimespec.Spec, error) {
	data, err := ioutil.ReadFile(filepath.Join(state.Dir(id), ConfigFileName))
	if err != nil {
		return nil, err
	}
	config := &runtimespec.Spec{}
	err = json.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}
	return config, nil
}
