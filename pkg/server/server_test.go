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
	"os"
	"reflect"
	"syscall"
	"testing"

	"github.com/nbena/gotask/pkg/req"

	"github.com/nbena/gotask/pkg/task"
)

type serverTestCase struct {
	server  *TaskServer
	tasks   []task.Task
	outputs []string
	config  *Config
	client  *http.Client
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

func (s *serverTestCase) request(method, postfix string, expectedStatus int, t *testing.T) *http.Response {

	uri := fmt.Sprintf("http://%s:%d/%s", s.config.ListenAddr, s.config.ListenPort, postfix)

	req, err := http.NewRequest(method, uri, nil)
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

	resp := s.request("GET", "refresh", http.StatusNoContent, t)

	if resp == nil {
		t.Fatalf("Impossible to do the request\n")
	}
}

func (s *serverTestCase) list(print bool, t *testing.T) {

	resp := s.request("GET", "list", http.StatusOK, t)

	if resp == nil {
		t.Fatalf("Impossible to do the request\n")
	}

	body := resp.Body
	var receiver req.ListMessageResponse

	decoder := json.NewDecoder(body)
	if err := decoder.Decode(&receiver); err != nil {
		t.Errorf("Fail to parse /list response: %s\n", err.Error())
	}

	if print {
		fmt.Printf("%v\n", receiver.Tasks)
	}

	if !reflect.DeepEqual(receiver.Tasks, s.tasks) {
		t.Errorf("/list failed:\ngot: %v\nexpected: %v\n", receiver.Tasks, s.tasks)
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
					"ls . | grep file1",
				},
				ShowOutput: true,
				Long:       false,
				Shell:      "bash",
			},
		},
		outputs: []string{
			"hello",
			"",
			"file1\n",
		},
		config: &Config{
			ListenAddr:       "127.0.0.1",
			ListenPort:       7879,
			TaskFile:         "tasks.json",
			InternalChanSize: 5,
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
		testCase.server.serverCloseChan <- syscall.SIGINT
		testCase.server.taskManagerCloseChan <- syscall.SIGINT
		testCase.end(t)
	}
}
