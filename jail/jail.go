package jail

type Jail interface {
	// Attach attaches the current running process to the jail
	Attach() error
}

type jail struct {
	id ID
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
