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

	// RunjExtensionFileName is the name of an additional file, specifying only
	// the experimental FreeBSD section, which can be merged into the regular
	// bundle config.  This allows for software which generates a config file
	// unaware of FreeBSD and runj to be augmented by an additional program
	// that specifies additional settings.
	RunjExtensionFileName = "runj.ext.json"
)

// StoreConfig copies the config file provided in the input bundle to the state
// directory for the container.  The file must be copied to comply with this
// requirement from the OCI runtime specification:
// Any changes made to the config.json file after this operation will not have
// an effect on the container.
func StoreConfig(id, bundlePath string) error {
	err := copyFile(filepath.Join(bundlePath, ConfigFileName), filepath.Join(state.Dir(id), ConfigFileName))
	if err != nil {
		return err
	}
	extFilename := filepath.Join(bundlePath, RunjExtensionFileName)
	if _, err = os.Stat(extFilename); err == nil {
		err = copyFile(extFilename, filepath.Join(state.Dir(id), RunjExtensionFileName))
		if err != nil {
			return err
		}
	}
	return nil
}

func copyFile(source, dest string) error {
	input, err := os.OpenFile(source, os.O_RDONLY, 0)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
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
	if _, err = os.Stat(filepath.Join(state.Dir(id), RunjExtensionFileName)); err == nil {
		extData, err := ioutil.ReadFile(filepath.Join(state.Dir(id), RunjExtensionFileName))
		if err != nil {
			return nil, err
		}
		freebsd := &runtimespec.FreeBSD{}
		err = json.Unmarshal(extData, freebsd)
		if err != nil {
			return nil, err
		}
		merge(config, freebsd)
	}
	return config, nil
}

// merge processes an existing spec and additional FreeBSD section to merge them
// together.  Fields specified in the original spec are preserved except in the
// case where they are overwritten.  Slices the FreeBSD section are appended to
// slices specified in the original spec.
func merge(spec *runtimespec.Spec, freebsd *runtimespec.FreeBSD) {
	if spec == nil || freebsd == nil {
		return
	}
	if spec.FreeBSD == nil {
		spec.FreeBSD = &runtimespec.FreeBSD{}
	}
	if freebsd.Network != nil {
		if spec.FreeBSD.Network == nil {
			spec.FreeBSD.Network = &runtimespec.FreeBSDNetwork{}
		}
		if freebsd.Network.IPv4 != nil {
			if spec.FreeBSD.Network.IPv4 == nil {
				spec.FreeBSD.Network.IPv4 = &runtimespec.FreeBSDIPv4{}
			}
			if freebsd.Network.IPv4.Mode != "" {
				spec.FreeBSD.Network.IPv4.Mode = freebsd.Network.IPv4.Mode
			}
			if len(freebsd.Network.IPv4.Addr) > 0 {
				spec.FreeBSD.Network.IPv4.Addr = append(spec.FreeBSD.Network.IPv4.Addr, freebsd.Network.IPv4.Addr...)
			}
		}
	}
}
