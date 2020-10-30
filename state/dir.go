package state

import (
	"os"
	"path/filepath"
)

const (
	defaultStateDir = "/var/lib/runj/jails"
	stateDir        = defaultStateDir
)

func Dir(id string) string {
	return filepath.Join(stateDir, id)
}

func Remove(id string) error {
	return os.RemoveAll(Dir(id))
}
