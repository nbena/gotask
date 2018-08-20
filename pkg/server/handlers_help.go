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
	"fmt"
	"log"
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
	done := make(chan string)
	errChan := make(chan [2]string)
	runtimeTask.WaitPoll(id, done, errChan)

	msg := &req.ShortRunningTaskResponse{
		Command: strings.Join(runtimeTask.Args, ""),
	}

	select {
	case <-done:
		if runtimeTask.ShowOutput {
			out, err := runtimeTask.Output()
			if err != nil {
				msg.Error = fmt.Sprintf("Fail to get process output: %s", err.Error())
			} else {
				msg.Output = string(out)
			}
		}
	case errDesc := <-errChan:
		msg.Error = fmt.Sprintf("ERROR %s", errDesc[1])
	}

	return msg
}

func (t *TaskServer) handleCompletedPoll(
	taskID string,
	taskInfo task.RuntimeTaskInfo) *req.PollStatusCompletedResponse {
	var err error
	outStr := ""
	errStr := ""
	if taskInfo.ShowOutput {
		if outStr, err = taskInfo.StdoutStr(); err != nil {
			errStr = fmt.Sprintf("Fail to get STOUT: %s", err.Error())
			log.Printf("Error in get STDOUT: %s", err.Error())
		}
	}

	if errStr, err = taskInfo.StderrStr(); err != nil {
		log.Printf("Error in get STDERR: %s", err.Error())
	}
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
