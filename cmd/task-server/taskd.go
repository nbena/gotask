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
	"github.com/nbena/gotask/pkg/task"
)

// TaskServer is the HTTP server
type TaskServer struct {
	// server *http.ServerMux
	taskMap        TaskMap
	pendingTasks   longRunningTasksMap
	completedTasks longRunningTasksMap

	completeChan chan string
	errChan      chan [2]string

	config *RuntimeConfig

	taskManagerCloseChan chan struct{}
	serverCloseChan      chan struct{}
}

type taskID struct {
	task.Task
	ID string
}

func (t *TaskServer) moveTasks(id string, unlockCompleted bool) task.RuntimeTaskInfo {
	// first remove from the pending map
	t.pendingTasks.Lock()
	old := t.pendingTasks.taskMap[id]
	delete(t.pendingTasks.taskMap, id)
	t.pendingTasks.Unlock()

	// the move to the complete map
	t.completedTasks.Lock()
	t.completedTasks.taskMap[id] = old

	// the called may need to do other operation before unlock
	if unlockCompleted {
		t.completedTasks.Unlock()
	}

	return old
}

// this manages the different running processes
func (t *TaskServer) taskManager() {
	// TODO shutdown
	loop := true
	for loop {
		select {
		case id := <-t.completeChan:
			// ok the task is finished, we move it
			// to the completed tasks map
			t.moveTasks(id, true)

		case errDesc := <-t.errChan:

			old := t.moveTasks(errDesc[0], false)
			// we write the error to stderr
			old.Cmd.Stderr.Write([]byte("ERROR IN WAIT: " + errDesc[1]))
		// exit
		case <-t.taskManagerCloseChan:
			loop = false
		}
	}
}
