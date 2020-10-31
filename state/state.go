package state

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
)

const stateFile = "state.json"

type Status string

const (
	StatusCreating Status = "creating"
	StatusCreated  Status = "created"
	StatusRunning  Status = "running"
	StatusStopped  Status = "stopped"
)

type State struct {
	ID     string
	JID    int
	Status Status
	Bundle string
	PID    int
}

func Load(id string) (*State, error) {
	d, err := ioutil.ReadFile(filepath.Join(Dir(id), stateFile))
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

func (s *State) Save() error {
	f, err := ioutil.TempFile(Dir(s.ID), "state")
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
