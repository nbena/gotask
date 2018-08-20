package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nbena/gotask/pkg/task"
)

func writeError(w http.ResponseWriter, msg string, addContentType bool,
	status int) {
	if addContentType {
		w.Header().Add("Content-type", "application/json, charset=utf-8")
	}
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	encoder.Encode(task.ErrorMessageResponse{
		Error: msg,
	})
}

// refresh
func (t *TaskServer) renew(w http.ResponseWriter, r *http.Request) {

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
	receiver := make([]taskID, len(t.taskMap.tasks))

	i := 0
	for key, value := range t.taskMap.tasks {
		task := taskID{
			Task: value,
			ID:   key,
		}
		receiver[i] = task
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
	var req task.ExecuteMessageRequest
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

		t.completedTasks.RLock()
		taskInfo, ok = t.completedTasks.taskMap[taskID]
		t.completedTasks.RUnlock()
	} else {
		inPending = true
	}

	if !ok {
		writeError(w, fmt.Sprintf("Task %s not found", taskID), false, http.StatusNotFound)
		return
	}

	var msg interface{}

	if inPending {
		msg = &task.PollStatusInProgressResponse{
			ID:     taskID,
			Status: task.PollStatusInProgress,
		}
	} else {
		msg = t.handleCompletedPoll(taskID, taskInfo)
	}

	encoder := json.NewEncoder(w)
	encoder.Encode(msg)
}
