package jail

import (
	"errors"
	"sync"
)

// Jail represents an existing jail
type Jail interface {
	// Attach attaches the current running process to the jail
	Attach() error
	// Remove destroys the jail
	Remove() error
}

type jail struct {
	m       sync.Mutex
	id      ID
	removed bool
}

// FromName queries the OS for a jail with the specified name
func FromName(name string) (Jail, error) {
	id, err := find(name)
	if err != nil {
		return nil, err
	}
	return &jail{id: id}, nil
}

// Attach attaches the current running process to the jail
func (j *jail) Attach() error {
	return attach(j.id)
}

func (j *jail) Remove() error {
	j.m.Lock()
	defer j.m.Unlock()
	if j.removed {
		return errors.New("already removed")
	}

	if err := remove(j.id); err != nil {
		return err
	}
	j.removed = true
	return nil
}
