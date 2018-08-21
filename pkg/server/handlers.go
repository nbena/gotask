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
	"net/http"

	"github.com/nbena/gotask/pkg/req"
	"github.com/nbena/gotask/pkg/task"
)

func writeError(w http.ResponseWriter, msg string, addContentType bool,
	status int) {
	if addContentType {
		w.Header().Add("Content-type", "application/json, charset=utf-8")
	}
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	encoder.Encode(req.ErrorMessageResponse{
		Error: msg,
	})
}

// refresh
func (t *TaskServer) refresh(w http.ResponseWriter, r *http.Request) {

	if err := t.taskMap.ReadTasks(t.config.taskFilePath, true); err != nil {
		writeError(w, err.Error(), true, http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

// list
func (t *TaskServer) list(w http.ResponseWriter, r *http.Request) {

	w.Header().Add("Content-type", "application/json, charset=utf-8")
	w.WriteHeader(http.StatusNoContent)

	t.taskMap.RLock()
	receiver := req.NewListMessageResponse(len(t.taskMap.tasks))

	i := 0
	for _, value := range t.taskMap.tasks {
		receiver.Tasks[i] = value
		i++
	}

	encoder := json.NewEncoder(w)
	encoder.Encode(receiver)
	t.taskMap.RUnlock()
}

// execute
func (t *TaskServer) execute(w http.ResponseWriter, r *http.Request) {

	w.Header().Add("Content-type", "application/json, charset=utf-8")

	decoder := json.NewDecoder(r.Body)
	var req req.ExecuteMessageRequest
	if err := decoder.Decode(&req); err != nil {
		writeError(w, err.Error(), false, http.StatusInternalServerError)
		return
	}

	// now looking for the task in the map
	t.taskMap.RLock()
	taskToRun, ok := t.taskMap.tasks[req.TaskName]
	if !ok {
		writeError(w, fmt.Sprintf("Task %s not found", req.TaskName),
			false, http.StatusNotFound)
		return
	}
	t.taskMap.RUnlock()

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

	encoder := json.NewEncoder(w)
	encoder.Encode(msg)
}

// poll
func (t *TaskServer) poll(w http.ResponseWriter, r *http.Request) {

	w.Header().Add("Content-type", "application/json, charset=utf-8")

	q := r.URL.Query()
	taskID := q.Get("id")
	if taskID == "" {
		writeError(w, "URI not valid", false, http.StatusBadRequest)
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
		// t.completedTasks.RUnlock()
	} else {
		inPending = true
	}

	if !ok {
		writeError(w, fmt.Sprintf("Task %s not found", taskID), false, http.StatusNotFound)
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

	encoder := json.NewEncoder(w)
	encoder.Encode(msg)
}
