package jail

import (
	"io"
	"os"
	"path/filepath"

	"github.com/containerd/containerd/mount"

	"go.sbk.wtf/runj/runtimespec"
)

// Mount mounts the mounts
func Mount(id string, ociConfig *runtimespec.Spec) error {
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
			if !stat.IsDir() {
				// Save the original file so that we can approximate Linux
				// bind file mounts
				saveFile(id, dest)
				copyFile(id, m.Source, dest)
				continue
			}
		} else {
			err = createIfNotExists(dest, true)
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
	return nil
}

func saveDir(id, dest string) string {
	return filepath.Join(filepath.Dir(dest), ".save-"+id)
}

func saveFile(id, dest string) error {
	_, err := os.Stat(dest)
	if err == nil {
		save := saveDir(id, dest)
		if err := os.MkdirAll(save, 0700); err != nil {
			return err
		}
		if err := os.Rename(dest, filepath.Join(save, filepath.Base(dest))); err != nil {
			return err
		}
	}
	return nil
}

func restoreFile(id, dest string) error {
	save := filepath.Join(saveDir(id, dest), filepath.Base(dest))
	_, err := os.Stat(save)
	if err == nil {
		if err := os.Rename(save, dest); err != nil {
			return err
		}
	}
	return nil

}

func copyFile(id, source, dest string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}

// Unmount attempts to unmount all mounts present in the spec.  If multiple
// errors occur, Unmount returns the first.
func Unmount(id string, ociConfig *runtimespec.Spec) error {
	var retErr error
	saveDirs := make(map[string]bool)
	for i := len(ociConfig.Mounts) - 1; i >= 0; i-- {
		m := ociConfig.Mounts[i]
		dest := filepath.Join(ociConfig.Root.Path, m.Destination)
		if m.Type == "nullfs" {
			stat, err := os.Stat(m.Source)
			if err != nil {
				return err
			}
			if !stat.IsDir() {
				if err := os.Remove(dest); err != nil {
					return err
				}
				saveDirs[saveDir(id, dest)] = true
				restoreFile(id, dest)
				continue
			}
		}
		err := mount.Unmount(dest, 0)
		if err != nil && retErr == nil {
			retErr = err
		}
	}
	for saveDir := range saveDirs {
		if err := os.RemoveAll(saveDir); err != nil {
			return err
		}
	}
	return retErr
}
