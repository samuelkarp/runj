package jail

import (
	"path/filepath"

	"github.com/containerd/containerd/mount"

	"go.sbk.wtf/runj/runtimespec"
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
			m.Source = "null"
		}
		dest := filepath.Join(ociConfig.Root.Path, ociMount.Destination)
		err = m.Mount(dest)
		if err != nil {
			return err
		}
		unwind = append(unwind, dest)
	}
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
