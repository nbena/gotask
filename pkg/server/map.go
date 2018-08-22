// go-task, a simple client-server task runner
// Copyright (C) 2018 nbena
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/nbena/gotask/pkg/task"
)

// uniqueID generated a pseudo-random ID
// that is guaranteed to be unique for the given
// map
func uniqueID(toCheck map[string]task.RuntimeTaskInfo) string {
	source := make([]byte, 40)
	var id string
	var err error
	loop := true

	for loop {
		_, err = rand.Read(source)
		if err != nil {
			id := hex.EncodeToString(source)
			if _, ok := toCheck[id]; !ok {
				loop = false
			}
		}
	}
	return id
}

func uniqueID2() string {
	loop := true
	var result string
	source := make([]byte, 40)
	for loop {
		_, err := rand.Read(source)
		if err == nil {
			result = hex.EncodeToString(source)
			loop = false
		}
	}
	return result
}

// map used for long-running tasks
type longRunningTasksMap struct {
	taskMap map[string]task.RuntimeTaskInfo
	*sync.RWMutex
}

type taskMap struct {
	*sync.Map
}

// ReadTasks fills the map. If empty is true, the map is emptied
// before the new filling process.
func (m *taskMap) ReadTasks(path string, empty bool) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var receiver []task.Task

	if err = decoder.Decode(&receiver); err != nil {
		return err
	}

	if empty {
		m.Map = &sync.Map{}
	}

	for _, toAdd := range receiver {
		if _, loaded := m.LoadOrStore(toAdd.Name, toAdd); loaded {
			err = fmt.Errorf("Task already present: %s", toAdd.Name)
			break
		}
	}
	return err
}

// Write writes the map to file.
func (m *taskMap) Write(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	var receiver []task.Task
	m.Range(func(key, value interface{}) bool {
		receiver = append(receiver, value.(task.Task))
		return true
	})
	data, err := json.Marshal(receiver)
	if err != nil {
		return err
	}

	if _, err = file.Write(data); err != nil {
		return err
	}
	return nil
}
