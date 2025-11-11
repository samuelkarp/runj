package jail

import (
	"errors"
	"fmt"
	"sync"
)

// Jail represents an existing jail
type Jail interface {
	// Attach attaches the current running process to the jail
	Attach() error
	// Remove destroys the jail
	Remove() error
	id() ID
}

type jail struct {
	m       sync.Mutex
	_id     ID
	removed bool
}

func Create(config *CreateParams) (Jail, error) {
	iovec, err := config.iovec()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize iovec: %w", err)
	}
	jid, err := set(iovec, _FLAG_CREATE)
	if err != nil {
		return nil, fmt.Errorf("failed to invoke jail_set: %w", err)
	}
	return &jail{_id: jid}, nil
}

// FromName queries the OS for a jail with the specified name
func FromName(name string) (Jail, error) {
	id, err := find(name)
	if err != nil {
		return nil, err
	}
	return &jail{_id: id}, nil
}

// Attach attaches the current running process to the jail
func (j *jail) Attach() error {
	return attach(j._id)
}

func (j *jail) Remove() error {
	j.m.Lock()
	defer j.m.Unlock()
	if j.removed {
		return errors.New("already removed")
	}

	if err := remove(j._id); err != nil {
		return err
	}
	j.removed = true
	return nil
}

func (j *jail) id() ID {
	return j._id
}
