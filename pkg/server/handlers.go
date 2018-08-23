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
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/nbena/gotask/pkg/req"
	"github.com/nbena/gotask/pkg/task"
)

// refresh
func (t *TaskServer) refresh(w http.ResponseWriter, r *http.Request) {

	if ok := checkMethod(http.MethodPost, w, r); !ok {
		return
	}

	if err := t.taskMap.ReadTasks(t.config.taskFilePath, true); err != nil {
		writeError(w, err.Error(), true, http.StatusInternalServerError)
	} else {
		w.WriteHeader(StatusRefresh)
	}
}

// list
func (t *TaskServer) list(w http.ResponseWriter, r *http.Request) {

	if ok := checkMethod(http.MethodGet, w, r); !ok {
		return
	}

	var tasks []task.Task
	t.taskMap.Range(func(key, value interface{}) bool {
		tasks = append(tasks, value.(task.Task))
		return true
	})
	receiver := req.ListMessageResponse{
		Tasks: tasks,
	}

	encodeWithError(w, StatusList, receiver)
}

// execute
func (t *TaskServer) execute(w http.ResponseWriter, r *http.Request) {

	if ok := checkMethod(http.MethodPut, w, r); !ok {
		return
	}

	decoder := json.NewDecoder(r.Body)
	var req req.ExecuteMessageRequest
	if err := decoder.Decode(&req); err != nil {
		writeError(w, err.Error(), false, http.StatusInternalServerError)
		return
	}

	// now looking for the task in the map
	toRun, ok := t.taskMap.Load(req.TaskName)
	if !ok {
		writeError(w, fmt.Sprintf("Task %s not found", req.TaskName),
			true, http.StatusNotFound)
		return
	}

	taskToRun := toRun.(task.Task)

	// now run the fucking task.
	runtimeTask, err := taskToRun.Run()
	if err != nil {
		writeError(w, fmt.Sprintf("Running error: %s", err.Error()),
			false, http.StatusInternalServerError)
		return
	}

	var msg interface{}

	if taskToRun.Long {
		// THE WAITING GOROUTINE is not started till
		// you poll for it
		msg = t.handleExecuteLongTask(*runtimeTask)
	} else {
		msg = t.handleExecuteShortTask(*runtimeTask)
	}

	encodeWithError(w, StatusExecute, msg)
}

// poll
func (t *TaskServer) poll(w http.ResponseWriter, r *http.Request) {

	if ok := checkMethod(http.MethodGet, w, r); !ok {
		return
	}

	q := r.URL.Query()
	taskID := q.Get("id")
	if taskID == "" {
		writeError(w, "URI not valid", true, http.StatusBadRequest)
		return
	}

	var taskInfo task.RuntimeTaskInfo
	var ok, inPending bool

	inPending = false
	t.pendingTasks.RLock()
	taskInfo, ok = t.pendingTasks.taskMap[taskID]
	t.pendingTasks.RUnlock()
	if !ok {

		// we need a complete lock over the completed tasks map
		// because we'll then need to remove it.
		t.completedTasks.Lock()
		taskInfo, ok = t.completedTasks.taskMap[taskID]
	} else {
		inPending = true
	}

	if !ok {
		writeError(w, fmt.Sprintf("Task %s not found", taskID), true, http.StatusNotFound)
		t.completedTasks.Unlock()
		return
	}

	var msg interface{}

	if inPending {
		msg = &req.PollStatusInProgressResponse{
			ID:     taskID,
			Status: req.PollStatusInProgress,
		}
	} else {
		msg = t.handleCompletedPoll(taskID, taskInfo)
		t.completedTasks.Unlock()
	}

	encodeWithError(w, StatusPoll, msg)
}

// update
func (t *TaskServer) addOrModify(w http.ResponseWriter, r *http.Request) {
	if ok := checkMethod(MethodAddModify, w, r); !ok {
		return
	}

	addTaskReq := req.AddTaskRequest{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&addTaskReq); err != nil {
		writeError(w, err.Error(), true, http.StatusInternalServerError)
		return
	}

	_, loaded := t.taskMap.LoadOrStore(addTaskReq.Task.Name, addTaskReq.Task)
	if loaded {
		// if alreadt there, modify
		t.taskMap.Store(addTaskReq.Task.Name, addTaskReq.Task)
	}

	// t.taskMap.Store()
	go func() {
		if err := t.taskMap.Write(t.config.taskFilePath); err != nil {
			log.Printf("Error write to file: %s\n", err.Error())
		}
	}()

	w.WriteHeader(StatusAddModify)
}
