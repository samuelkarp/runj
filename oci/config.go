package oci

import (
	"encoding/json"
	"os"
	"path/filepath"

	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
	"go.sbk.wtf/runj/internal/util"
	runjspec "go.sbk.wtf/runj/runtimespec"
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
	err := util.CopyFile(filepath.Join(bundlePath, ConfigFileName), filepath.Join(state.Dir(id), ConfigFileName), 0600)
	if err != nil {
		return err
	}
	extFilename := filepath.Join(bundlePath, RunjExtensionFileName)
	if _, err = os.Stat(extFilename); err == nil {
		err = util.CopyFile(extFilename, filepath.Join(state.Dir(id), RunjExtensionFileName), 0600)
		if err != nil {
			return err
		}
	}
	return nil
}

// LoadConfig loads the config file stored in the state directory
func LoadConfig(id string) (*runtimespec.Spec, error) {
	data, err := os.ReadFile(filepath.Join(state.Dir(id), ConfigFileName))
	if err != nil {
		return nil, err
	}
	config := &runtimespec.Spec{}
	err = json.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}
	if _, err = os.Stat(filepath.Join(state.Dir(id), RunjExtensionFileName)); err == nil {
		extData, err := os.ReadFile(filepath.Join(state.Dir(id), RunjExtensionFileName))
		if err != nil {
			return nil, err
		}
		freebsd := &runjspec.FreeBSD{}
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
func merge(spec *runtimespec.Spec, freebsd *runjspec.FreeBSD) {
	if spec == nil || freebsd == nil {
		return
	}
	if spec.FreeBSD == nil {
		spec.FreeBSD = &runtimespec.FreeBSD{}
	}
	if spec.FreeBSD.Jail == nil {
		spec.FreeBSD.Jail = &runtimespec.FreeBSDJail{}
	}
	if freebsd.Network != nil {
		if freebsd.Network.IPv4 != nil {
			if freebsd.Network.IPv4.Mode != "" {
				spec.FreeBSD.Jail.Ip4 = runtimespec.FreeBSDSharing(freebsd.Network.IPv4.Mode)
			}
			if len(freebsd.Network.IPv4.Addr) > 0 {
				spec.FreeBSD.Jail.Ip4Addr = append(spec.FreeBSD.Jail.Ip4Addr, freebsd.Network.IPv4.Addr...)
			}
		}
		if freebsd.Network.VNet != nil {
			if freebsd.Network.VNet.Mode != "" {
				spec.FreeBSD.Jail.Vnet = runtimespec.FreeBSDSharing(freebsd.Network.VNet.Mode)
			}
			if len(freebsd.Network.VNet.Interfaces) > 0 {
				spec.FreeBSD.Jail.VnetInterfaces = append(spec.FreeBSD.Jail.VnetInterfaces, freebsd.Network.VNet.Interfaces...)
			}
		}
	}
}
