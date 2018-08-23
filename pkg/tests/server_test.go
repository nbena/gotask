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

package tests

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
	"github.com/nbena/gotask/pkg/server"

	"github.com/nbena/gotask/pkg/task"
)

type serverTestCase struct {
	server  *server.TaskServer
	tasks   []task.Task
	outputs []string
	config  *server.Config
	client  *http.Client

	toAddConflict task.Task
	toAdd         task.Task
	toMod         task.Task
}

func basicServerRun(config *server.Config, tasks []task.Task) (*server.TaskServer, error) {
	// we write tasks and config to file
	taskFile, err := os.Create(config.TaskFile)
	if err != nil {
		return nil, err
	}

	encoder := json.NewEncoder(taskFile)
	if encoder.Encode(tasks) != nil {
		return nil, err
	}

	taskFile.Close()

	server, err := server.NewServer(config)
	if err != nil {
		return nil, err
	}
	return server, nil
}

func (s *serverTestCase) startServer() error {

	server, err := basicServerRun(s.config, s.tasks)
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

func (s *serverTestCase) request(method, postfix string,
	expectedStatus int, body io.ReadCloser,
	t *testing.T) *http.Response {

	uri := fmt.Sprintf("http://%s:%d%s", s.config.ListenAddr, s.config.ListenPort, postfix)

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

	resp := s.request(server.MethodRefresh, server.APIRefresh, server.StatusRefresh, nil, t)

	if resp == nil {
		t.Fatalf("Impossible to do the request\n")
	}
}

func (s *serverTestCase) list(print bool, t *testing.T) []task.Task {

	resp := s.request(server.MethodList, server.APIList, server.StatusList, nil, t)

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

	tasksCheck(s.tasks, receiver.Tasks, t)

	return receiver.Tasks
}

func (s *serverTestCase) poll(id string, expectedStatus int, t *testing.T) {

	resp := s.request(server.MethodPoll, server.APIPoll+"?id="+id, expectedStatus, nil, t)
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

	resp := s.request(server.MethodExecute, server.APIExecute, server.StatusExecute, reqBody, t)
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

func (s *serverTestCase) internalAdd(toAdd task.Task, t *testing.T) task.Task {
	data := req.AddTaskRequest{
		Task: toAdd,
	}

	dataEnc, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Fail to marshal data: %s\n", err.Error())
	}
	reqBody := ioutil.NopCloser(bytes.NewReader(dataEnc))

	s.request(server.MethodAddModify, server.APIAddModify, server.StatusAddModify, reqBody, t)

	time.Sleep(15 * time.Millisecond)

	tasks := s.list(true, t)
	if tasks == nil {
		return task.Task{}
	}

	tasksIn(toAdd, tasks, t)

	return toAdd
}

func (s *serverTestCase) add(t *testing.T) {

	s.internalAdd(s.toAdd, t)

	mod := s.internalAdd(s.toMod, t)
	if mod.Name == s.toMod.Name {
		if reflect.DeepEqual(mod, s.toAdd) {
			t.Errorf("AddMod failed\ngot: %v\nexp: %v", mod, s.toAdd)
		}
	} else {
		t.Errorf("The two mod are not the same:\ngot: %v\nexp: %v\n",
			mod, s.toMod)
	}
}

var serverTests = []serverTestCase{
	{
		tasks: tasks,
		outputs: []string{
			"hello",
			"",
			"file1\n",
			"",
		},
		config: &server.Config{
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
		toAdd: taskToAdd,
		toMod: taskToMod,
	},
}

func (s *serverTest) runTest(t *testing.T) {
	for _, testCase := range s.tests {
		if err := testCase.startServer(); err != nil {
			t.Errorf("Fail to start server: %s\n", err.Error())
		}
		testCase.refresh(t)
		testCase.list(true, t)
		for i := range testCase.tasks {
			testCase.execute(i, t)
		}
		testCase.add(t)
		testCase.server.ServerCloseChan <- syscall.SIGINT
		end(testCase.config, t)
	}
}

func TestServer(t *testing.T) {
	tests := &serverTest{
		tests: serverTests,
	}
	tests.runTest(t)
}
