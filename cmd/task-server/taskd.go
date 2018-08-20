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
