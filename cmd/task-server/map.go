package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"sync"

	"github.com/nbena/gotask/pkg/task"
)

// TaskMap wraps a map and RW lock.
type TaskMap struct {
	tasks map[string]task.Task
	*sync.RWMutex
}

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

// ReadTasks (re)fill the current map.
// if empty, a new hash table will be created.
func (t *TaskMap) ReadTasks(path string, empty bool) error {

	file, err := os.Open(path)
	defer file.Close()

	if err != nil {
		return err
	}

	var receiver []task.Task
	decoder := json.NewDecoder(file)

	if err = decoder.Decode(receiver); err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()

	if empty {
		t.tasks = make(map[string]task.Task)
	}

	for _, task := range receiver {
		t.tasks[task.Name] = task
	}

	return nil
}

// map used for long-running tasks
type longRunningTasksMap struct {
	taskMap map[string]task.RuntimeTaskInfo
	*sync.RWMutex
}
