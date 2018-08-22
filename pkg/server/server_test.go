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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/nbena/gotask/pkg/req"

	"github.com/nbena/gotask/pkg/task"
)

type serverTestCase struct {
	server  *TaskServer
	tasks   []task.Task
	outputs []string
	config  *Config
	client  *http.Client

	toAddConflict task.Task
	toAdd         task.Task
}

// type serverTestCase struct {
// 	server *TaskServer
// }

func (s *serverTestCase) startServer() error {

	// we write tasks and config to file
	taskFile, err := os.Create(s.config.TaskFile)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(taskFile)
	if encoder.Encode(s.tasks) != nil {
		return err
	}

	taskFile.Close()

	server, err := NewServer(s.config)
	if err != nil {
		return err
	}

	s.client = &http.Client{}

	s.server = server
	go func() {
		s.server.Run()
	}()

	return nil
}

func (s *serverTestCase) end(t *testing.T) {
	if err := os.Remove(s.config.TaskFile); err != nil {
		t.Errorf("Error in deleting task file: %s\n", s.config.TaskFile)
	}
}

func (s *serverTestCase) request(method, postfix string,
	expectedStatus int, body io.ReadCloser,
	t *testing.T) *http.Response {

	uri := fmt.Sprintf("http://%s:%d/%s", s.config.ListenAddr, s.config.ListenPort, postfix)

	req, err := http.NewRequest(method, uri, body)
	if err != nil {
		t.Errorf("Error in preparing request: %s\n", err.Error())
		return nil
	}

	resp, err := s.client.Do(req)
	if err != nil {
		t.Errorf("Error in %s %s: %s\n", method, uri, err.Error())
		return nil
	}

	if resp.StatusCode != expectedStatus {
		t.Errorf("Status error:\ngot: %d\nexpected: %d\n", resp.StatusCode, expectedStatus)
		return nil
	}

	return resp

}

func (s *serverTestCase) refresh(t *testing.T) {

	resp := s.request("GET", "refresh", http.StatusNoContent, nil, t)

	if resp == nil {
		t.Fatalf("Impossible to do the request\n")
	}
}

func (s *serverTestCase) list(print bool, t *testing.T) []task.Task {

	resp := s.request("GET", "list", http.StatusOK, nil, t)

	if resp == nil {
		t.Fatalf("Impossible to do the request\n")
	}

	body := resp.Body
	defer resp.Body.Close()
	var receiver req.ListMessageResponse

	decoder := json.NewDecoder(body)
	if err := decoder.Decode(&receiver); err != nil {
		t.Errorf("Fail to parse /list response: %s\n", err.Error())
	}

	if print {
		fmt.Printf("%v\n", receiver.Tasks)
	}

	count := 0
	for _, task := range s.tasks {
		for _, gotTask := range receiver.Tasks {
			if reflect.DeepEqual(task, gotTask) {
				count++
			}
		}
	}
	// if !reflect.DeepEqual(receiver.Tasks, s.tasks) {
	// 	t.Errorf("/list failed:\ngot: %v\nexpected: %v\n", receiver.Tasks, s.tasks)
	// }
	if count != len(s.tasks) {
		t.Errorf("/list failed:\ngot: %v\nexpected: %v\n", receiver.Tasks, s.tasks)
	}

	return receiver.Tasks
}

func (s *serverTestCase) poll(id string, expectedStatus int, t *testing.T) {

	resp := s.request("GET", "poll?id="+id, expectedStatus, nil, t)
	if resp == nil {
		t.Fatalf("Impossible to do the request\n")
	}

	if expectedStatus == http.StatusNotFound {
		return
	}

	respBody := resp.Body
	defer respBody.Close()

	data, err := ioutil.ReadAll(respBody)
	if err != nil {
		t.Errorf("Fail to get the body: %s\n", err.Error())
		return
	}

	var receiver interface{}
	if strings.Index(string(data), "Completed") != -1 {
		receiver = req.PollStatusCompletedResponse{}
		if err := json.Unmarshal(data, &receiver); err != nil {
			t.Errorf("Error in completed unmarshal: %s\n", err.Error())
		}
	} else {
		receiver = req.PollStatusInProgressResponse{}
		if err := json.Unmarshal(data, &receiver); err != nil {
			t.Errorf("Error in in-progress unmarshal: %s\n", err.Error())
		}

		time.Sleep(100 * time.Millisecond)
		// wait and re-query
		s.poll(id, expectedStatus, t)
	}

	t.Logf("Response:\n%v\n", receiver)
}

func (s *serverTestCase) execute(i int, t *testing.T) {

	data := req.ExecuteMessageRequest{
		TaskName: s.tasks[i].Name,
	}

	dataEnc, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Fail to marshal data: %s\n", err.Error())
	}

	reqBody := ioutil.NopCloser(bytes.NewReader(dataEnc))

	resp := s.request("PUT", "exec", http.StatusOK, reqBody, t)
	if resp == nil {
		t.Fatalf("Impossible to do the request\n")
	}

	respBody := resp.Body

	respData, err := ioutil.ReadAll(respBody)
	defer respBody.Close()

	if err != nil {
		t.Errorf("Fail to get the body: %s\n", err.Error())
		return
	}

	if s.tasks[i].Long {

		var receiver req.LongRunningTaskResponse
		if err := json.Unmarshal(respData, &receiver); err != nil {
			t.Errorf("Fail to unmarshal data: %s\n", err.Error())
			return
		}

		t.Logf("Long task ID: %s\n", receiver.ID)

		// s.poll(receiver.ID, http.StatusOK, t)
	} else {

		var receiver req.ShortRunningTaskResponse
		if err := json.Unmarshal(respData, &receiver); err != nil {
			t.Errorf("Fail to unmarshal data: %s\n", err.Error())
			return
		}

		if receiver.Error != "" {
			t.Logf("Command ended with error: %s\n", receiver.Error)
		}

		if receiver.Output != s.outputs[i] {
			t.Errorf("Mismatched output:\ngot: %s\nexpected: %s\n",
				receiver.Output, s.outputs[i])
		}
	}
}

func (s *serverTestCase) add(expectedStatus int, t *testing.T) {

	toAdd := task.Task{}
	if expectedStatus == http.StatusConflict {
		toAdd = s.toAddConflict
	} else if expectedStatus == http.StatusNoContent {
		toAdd = s.toAdd
	}

	data := req.AddTaskRequest{
		Task: toAdd,
	}

	dataEnc, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Fail to marshal data: %s\n", err.Error())
	}
	reqBody := ioutil.NopCloser(bytes.NewReader(dataEnc))

	s.request("post", "add", expectedStatus, reqBody, t)

	if expectedStatus == http.StatusConflict {
		// nothing more to do
		return
	}

	// now issuing a list
	// but before wait.
	time.Sleep(15 * time.Millisecond)
	tasks := s.list(true, t)

	found := false
	for _, gotTask := range tasks {
		if gotTask.Name == s.toAdd.Name {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Task add but not found")
	}
}

var allTests = []serverTestCase{
	{
		tasks: []task.Task{
			{
				Name: "task1",
				Command: []string{
					"echo",
					"-n",
					"hello",
				},
				ShowOutput: true,
				Long:       false,
				Shell:      "bash",
			}, {
				Name: "task2",
				Command: []string{
					"touch",
					"file1",
				},
				ShowOutput: false,
				Long:       true,
				Shell:      "bash",
			}, {
				Name: "task3",
				Command: []string{
					"ls | grep file1",
				},
				ShowOutput: true,
				Long:       false,
				Shell:      "bash",
			}, {
				Name: "task4",
				Command: []string{
					"rm file1",
				},
				ShowOutput: false,
				Long:       false,
				Shell:      "bash",
			},
		},
		outputs: []string{
			"hello",
			"",
			"file1\n",
			"",
		},
		config: &Config{
			ListenAddr:       "127.0.0.1",
			ListenPort:       7879,
			TaskFile:         "tasks.json",
			InternalChanSize: 5,
		},
		toAddConflict: task.Task{
			Name: "task4",
			Command: []string{
				"echo",
				"go",
			},
			ShowOutput: true,
			Long:       false,
			Shell:      "bash",
		},
		toAdd: task.Task{
			Name: "task5",
			Command: []string{
				"echo",
				"go",
			},
			ShowOutput: true,
			Long:       false,
			Shell:      "bash",
		},
	},
}

func TestServer(t *testing.T) {
	for _, testCase := range allTests {
		if err := testCase.startServer(); err != nil {
			t.Errorf("Fail to start server: %s\n", err.Error())
		}
		testCase.refresh(t)
		testCase.list(true, t)
		for i := range testCase.tasks {
			t.Logf("Starting execute req for: %s\n", testCase.tasks[i].Name)
			testCase.execute(i, t)
		}
		testCase.add(http.StatusNoContent, t)
		testCase.server.serverCloseChan <- syscall.SIGINT
		testCase.server.taskManagerCloseChan <- syscall.SIGINT
		testCase.end(t)
	}
}
