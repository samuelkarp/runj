package jail

import (
	"os"
	"path/filepath"

	"github.com/containerd/containerd/mount"

	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
)

// Mount mounts the mounts
func Mount(ociConfig *runtimespec.Spec) error {
	var err error
	unwind := make([]string, 0)
	defer func() {
		if err == nil {
			return
		}
		for i := len(unwind) - 1; i >= 0; i-- {
			mount.Unmount(unwind[i], 0)
		}
	}()
	for _, ociMount := range ociConfig.Mounts {
		m := &mount.Mount{
			Type:    ociMount.Type,
			Source:  ociMount.Source,
			Options: ociMount.Options,
		}
		if m.Source == "" {
			// mount(8) requires a non-empty source
			m.Source = "null"
		}
		dest := filepath.Join(ociConfig.Root.Path, ociMount.Destination)
		if m.Type == "nullfs" {
			stat, err := os.Stat(m.Source)
			if err != nil {
				return err
			}
			err = createIfNotExists(dest, stat.IsDir())
			if err != nil {
				return err
			}
		}
		err = m.Mount(dest)
		if err != nil {
			return err
		}
		unwind = append(unwind, dest)
	}
	return nil
}

func createIfNotExists(path string, isDir bool) error {
	_, err := os.Stat(path)
	if err == nil {
		return nil
	}
	if !os.IsNotExist(err) {
		return nil
	}
	if isDir {
		return os.MkdirAll(path, 0755)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	f.Close()
	return nil
}

// Unmount attempts to unmount all mounts present in the spec.  If multiple
// errors occur, Unmount returns the first.
func Unmount(ociConfig *runtimespec.Spec) error {
	var retErr error
	for i := len(ociConfig.Mounts) - 1; i >= 0; i-- {
		dest := filepath.Join(ociConfig.Root.Path, ociConfig.Mounts[i].Destination)
		err := mount.Unmount(dest, 0)
		if err != nil && retErr == nil {
			retErr = err
		}
	}
	return retErr
}
