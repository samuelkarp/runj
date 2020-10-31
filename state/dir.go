package state

import (
	"os"
	"path/filepath"
)

const (
	defaultStateDir = "/var/lib/runj/jails"
	stateDir        = defaultStateDir
)

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
	err = s.Save()
	if err != nil {
		return nil, err
	}
	return s, nil
}

func Dir(id string) string {
	return filepath.Join(stateDir, id)
}

func Remove(id string) error {
	return os.RemoveAll(Dir(id))
}
