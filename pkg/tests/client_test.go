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
	"testing"

	"github.com/nbena/gotask/pkg/client"
	"github.com/nbena/gotask/pkg/server"
	"github.com/nbena/gotask/pkg/task"
)

type clientTestCase struct {
	taskToAdd    task.Task
	tasks        []task.Task
	client       *client.TaskClient
	server       *server.TaskServer
	clientConfig *client.Config
	serverConfig *server.Config
}

func (c *clientTestCase) startServer() error {
	server, err := basicServerRun(c.serverConfig, c.tasks)
	if err != nil {
		return err
	}
	c.server = server
	c.client = client.NewTaskClient(c.clientConfig)
	go func() {
		c.server.Run()
	}()

	return nil
}

func (c *clientTestCase) refresh(t *testing.T) {

	if err := c.client.Refresh(); err != nil {
		t.Errorf("Refresh error: %s\n", err.Error())
	}
}

func (c *clientTestCase) list(t *testing.T) {

	tasks, err := c.client.List()
	if err != nil {
		t.Errorf("List error: %s\n", err.Error())
	}

	tasksCheck(c.tasks, tasks, t)
}

func (c *clientTestCase) add(t *testing.T) {
	if err := c.client.Add(c.taskToAdd); err != nil {
		t.Errorf("Add error: %s\n", err.Error())
	}

	tasks, err := c.client.List()
	if err != nil {
		t.Errorf("List post-add error: %s\n", err.Error())
	}

	tasksCheck(append(c.tasks, c.taskToAdd), tasks, t)
}

func (c *clientTestCase) execute(i int, t *testing.T) {
	resp, err := c.client.Execute(c.tasks[i].Name)
	if err != nil {
		t.Errorf("Execute error: %s", err.Error())
	}

	t.Logf("Result: %v\n", resp)
}

func (c *clientTest) runTest(t *testing.T) {
	for _, testCase := range c.tests {
		if err := testCase.startServer(); err != nil {
			t.Errorf("Fail to start server: %s\n", err.Error())
		}
		testCase.refresh(t)
		testCase.list(t)
		for i := range testCase.tasks {
			testCase.execute(i, t)
		}
		testCase.add(t)
		end(testCase.serverConfig, t)
	}
}

var clientTests = []clientTestCase{
	{
		serverConfig: &server.Config{
			ListenAddr:       "127.0.0.1",
			ListenPort:       7667,
			TaskFile:         "tasks.json",
			InternalChanSize: 5,
		},
		tasks: tasks,
		clientConfig: &client.Config{
			ServerAddr: "127.0.0.1",
			ServerPort: 7667,
		},
		taskToAdd: taskToAdd,
	},
}

func TestClient(t *testing.T) {
	tests := &clientTest{
		tests: clientTests,
	}
	tests.runTest(t)
}
