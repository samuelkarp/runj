package state

import (
	"encoding/json"
	"os"
	"path/filepath"

	"go.sbk.wtf/runj/runtimespec"
)

const stateFile = "state.json"

// Status is the type for representing container status
type Status string

const (
	// StatusCreating represents a container in the process of being created
	StatusCreating Status = "creating"
	// StatusCreated represents a container that has been created but not started
	StatusCreated Status = "created"
	// StatusRunning represents a running container
	StatusRunning Status = "running"
	// StatusStopped represents a container that has exited
	StatusStopped Status = "stopped"
)

// State represents the state of a container
type State struct {
	// ID is the ID of the container
	ID string
	// JID is the jail ID of the jail backing the container
	JID int
	// Status is the status of the container
	Status Status
	// Bundle is the directory containing the config and rootfs
	Bundle string
	// PID is the primary process ID
	PID int
}

// Output is the expected output format for the state command
/*
{
    "ociVersion": "0.2.0",
    "id": "oci-container1",
    "status": "running",
    "pid": 4422,
    "bundle": "/containers/redis",
    "annotations": {
        "myKey": "myValue"
    }
}
*/
type Output struct {
	OCIVersion  string            `json:"ociVersion"`
	ID          string            `json:"id"`
	Status      string            `json:"status"`
	PID         int               `json:"pid,omitempty"`
	Bundle      string            `json:"bundle"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// Output converts the state to the "Output" format expected by hooks
func (s *State) Output() Output {
	return Output{
		OCIVersion: runtimespec.Version,
		ID:         s.ID,
		Status:     string(s.Status),
		PID:        s.PID,
		Bundle:     s.Bundle,
	}
}

// Load reads the state from disk and parses it
func Load(id string) (*State, error) {
	d, err := os.ReadFile(filepath.Join(Dir(id), stateFile))
	if err != nil {
		return nil, err
	}
	s := &State{}
	err = json.Unmarshal(d, s)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// initialize creates the original state file, checking for existence and
// failing if one already exists.  Initialize should be used as a guard to
// prevent overwriting a state file for an existing container.
func (s *State) initialize() error {
	_, err := os.OpenFile(filepath.Join(Dir(s.ID), stateFile), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	return s.Save()
}

// Save saves the state to disk
func (s *State) Save() error {
	f, err := os.CreateTemp(Dir(s.ID), "state")
	if err != nil {
		return err
	}
	defer func() {
		f.Close()
		if err != nil {
			os.Remove(f.Name())
		}
	}()
	d, err := json.Marshal(s)
	if err != nil {
		return err
	}
	_, err = f.Write(d)
	if err != nil {
		return err
	}
	os.Rename(f.Name(), filepath.Join(Dir(s.ID), stateFile))
	return nil
}
