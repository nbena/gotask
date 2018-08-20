package server

import (
	"fmt"
	"log"
	"strings"

	"github.com/nbena/gotask/pkg/task"
)

func (t *TaskServer) handleExecuteLongTask(runtimeTask task.RuntimeTaskInfo) *task.LongRunningTaskResponse {
	// grabbing a new ID for this runtime task
	t.pendingTasks.Lock()
	id := uniqueID(t.pendingTasks.taskMap)
	t.pendingTasks.taskMap[id] = runtimeTask
	t.pendingTasks.Unlock()

	return &task.LongRunningTaskResponse{
		Command: strings.Join(runtimeTask.Args, ""),
		ID:      id,
	}
}

func (t *TaskServer) handleExecuteShortTask(runtimeTask task.RuntimeTaskInfo) *task.ShortRunningTaskResponse {
	// wait for command to finish
	id := "not random ID"
	done := make(chan string)
	errChan := make(chan [2]string)
	runtimeTask.WaitPoll(id, done, errChan)

	msg := &task.ShortRunningTaskResponse{
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
	taskInfo task.RuntimeTaskInfo) *task.PollStatusCompletedResponse {
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
	return &task.PollStatusCompletedResponse{
		PollStatusInProgressResponse: task.PollStatusInProgressResponse{
			ID:     taskID,
			Status: task.PollStatusCompleted,
		},
		ShortRunningTaskResponse: task.ShortRunningTaskResponse{
			Command: strings.Join(taskInfo.Args, ""),
			Output:  outStr,
			Error:   errStr,
		},
	}
}
