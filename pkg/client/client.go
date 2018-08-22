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

package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/nbena/gotask/pkg/req"

	"github.com/nbena/gotask/pkg/server"
	"github.com/nbena/gotask/pkg/task"
)

// TaskClient is used to do request to the server.
type TaskClient struct {
	config *Config
	client *http.Client
}

// NewTaskClient returns a new TaskClient.
func NewTaskClient(config *Config) *TaskClient {

	return &TaskClient{
		config: config,
		// in future there will be more
		// options
		client: &http.Client{},
	}
}

func (c *TaskClient) request(method, postfix string,
	expectedStatus int, body io.ReadCloser) (*http.Response, error) {

	uri := fmt.Sprintf("http://%s:%d/%s",
		c.config.ServerAddr, c.config.ServerPort, postfix)

	req, err := http.NewRequest(method, uri, body)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != expectedStatus {
		return nil, err
	}

	return resp, nil
}

// List returns the list of tasks on the server.
func (c *TaskClient) List() ([]task.Task, error) {

	resp, err := c.request(server.MethodList, "list", server.StatusList, nil)
	if err != nil {
		return nil, err
	}

	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	rec := req.ListMessageResponse{}
	if err = json.Unmarshal(respData, &rec); err != nil {
		return nil, err
	}

	resp.Body.Close()

	return rec.Tasks, nil
}

// Refresh forces the server to re-read the tasks list.
func (c *TaskClient) Refresh() error {

	_, err := c.request(server.MethodRefresh, "refresh", server.StatusRefresh, nil)
	return err
}

// Add asks the server to add the task to the tasks list.
func (c *TaskClient) Add(toAdd task.Task) error {

	message := req.AddTaskRequest{
		Task: toAdd,
	}

	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	body := ioutil.NopCloser(bytes.NewBuffer(data))

	_, err = c.request(server.MethodAdd, "add", server.StatusAdd, body)
	return err
}

// Execute runs the task on the server.
func (c *TaskClient) Execute(taskName string) (*req.ShortRunningTaskResponse, error) {

	message := req.ExecuteMessageRequest{
		TaskName: taskName,
	}

	data, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}

	body := ioutil.NopCloser(bytes.NewBuffer(data))

	resp, err := c.request(server.MethodExecute, "exec", server.StatusExecute, body)
	if err != nil {
		return nil, err
	}

	// now we try to decode the response
	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	result := req.ShortRunningTaskResponse{}
	if strings.Index(string(respData), req.PollStatusInProgress) != -1 {
		// long task
		receiver := req.LongRunningTaskResponse{}
		if err = json.Unmarshal(respData, &receiver); err != nil {
			return nil, err
		}
		result, err = c.longTask(&receiver)
		if err != nil {
			return nil, err
		}
	} else {
		// short task
		err = json.Unmarshal(respData, &result)
		if err != nil {
			return nil, err
		}
	}
	return &result, nil
}

func (c *TaskClient) longTask(response *req.LongRunningTaskResponse) (req.ShortRunningTaskResponse, error) {

	// prepare a new request.
	result := req.ShortRunningTaskResponse{}
	loop := true
	for loop {
		// wait
		waiter := time.After(c.config.PollInterval)
		<-waiter
		// ok do the request
		resp, err := c.request(server.MethodPoll,
			fmt.Sprintf("poll?id=%s", response.ID), server.StatusPoll, nil)
		if err != nil {
			return req.ShortRunningTaskResponse{}, err
		}
		respData, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return req.ShortRunningTaskResponse{}, err
		}

		resp.Body.Close()
		if strings.Index(string(respData), req.PollStatusCompleted) != -1 {
			// completed
			err = json.Unmarshal(respData, &result)
			if err != nil {
				return req.ShortRunningTaskResponse{}, err
			}
			loop = false
		} // else still poll

	}
	return result, nil
}
