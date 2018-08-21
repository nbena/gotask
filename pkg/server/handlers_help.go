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
	"log"
	"net/http"
	"strings"

	"github.com/nbena/gotask/pkg/req"
	"github.com/nbena/gotask/pkg/task"
)

func (t *TaskServer) handleExecuteLongTask(runtimeTask task.RuntimeTaskInfo) *req.LongRunningTaskResponse {
	// grabbing a new ID for this runtime task
	t.pendingTasks.Lock()
	id := uniqueID(t.pendingTasks.taskMap)
	t.pendingTasks.taskMap[id] = runtimeTask
	t.pendingTasks.Unlock()

	return &req.LongRunningTaskResponse{
		Command: strings.Join(runtimeTask.Args, ""),
		ID:      id,
	}
}

func (t *TaskServer) handleExecuteShortTask(runtimeTask task.RuntimeTaskInfo) *req.ShortRunningTaskResponse {
	// wait for command to finish
	id := "not random ID"
	done := make(chan *task.CmdDoneChan)
	errChan := make(chan *task.CmdDoneChan)
	runtimeTask.WaitPoll(id, done, errChan)

	msg := &req.ShortRunningTaskResponse{
		Command: strings.Join(runtimeTask.Args, ""),
	}

	select {
	case res := <-done:
		if runtimeTask.ShowOutput {
			msg.Output = res.Output
		}
	case res := <-errChan:
		msg.Error = res.Error
	}

	return msg
}

func (t *TaskServer) handleCompletedPoll(
	taskID string,
	taskInfo task.RuntimeTaskInfo) *req.PollStatusCompletedResponse {
	outStr := ""
	errStr := ""
	if taskInfo.ShowOutput {
		// if outStr, err = taskInfo.StdoutStr(); err != nil {
		// 	errStr = fmt.Sprintf("Fail to get STOUT: %s", err.Error())
		// 	log.Printf("Error in get STDOUT: %s", err.Error())
		// }
		outStr = taskInfo.Output
	}
	errStr = taskInfo.Error

	delete(t.completedTasks.taskMap, taskID)

	return &req.PollStatusCompletedResponse{
		PollStatusInProgressResponse: req.PollStatusInProgressResponse{
			ID:     taskID,
			Status: req.PollStatusCompleted,
		},
		ShortRunningTaskResponse: req.ShortRunningTaskResponse{
			Command: strings.Join(taskInfo.Args, ""),
			Output:  outStr,
			Error:   errStr,
		},
	}
}

func encodeWithError(w http.ResponseWriter, okStatus int, input interface{}) {

	data, err := json.Marshal(input)
	if err != nil {
		writeError(w, err.Error(), true, http.StatusInternalServerError)
	} else {
		w.WriteHeader(okStatus)
		w.Header().Add("Content-type", "application/json, charset=utf-8")
		if _, err := w.Write(data); err != nil {
			log.Printf("Write error: %s\n", err.Error())
		}
	}
}

func writeError(w http.ResponseWriter, msg string, addContentType bool,
	status int) {
	w.WriteHeader(status)

	if addContentType {
		w.Header().Add("Content-type", "application/json, charset=utf-8")
	}

	encoder := json.NewEncoder(w)
	encoder.Encode(req.ErrorMessageResponse{
		Error: msg,
	})
}
