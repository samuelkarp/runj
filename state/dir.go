package state

import (
	"os"
	"path/filepath"
)

const defaultStateDir = "/var/lib/runj/jails"

// stateDir is the directory under which per-container state is stored.  It is a
// variable rather than a constant so that tests can redirect it to a temporary
// location.
var stateDir = defaultStateDir

// Create creates a state file for runj
func Create(id, bundle string) (*State, error) {
	s := &State{
		ID:     id,
		Bundle: bundle,
		Status: StatusCreating,
	}
	err := os.MkdirAll(Dir(id), 0755)
	if err != nil {
		return nil, err
	}
	err = s.initialize()
	if err != nil {
		return nil, err
	}
	return s, nil
}

// Dir returns the state directory for a container
func Dir(id string) string {
	return filepath.Join(stateDir, id)
}

// Remove removes the state for a container
func Remove(id string) error {
	return os.RemoveAll(Dir(id))
}
