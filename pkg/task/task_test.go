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

package task

import (
	"reflect"
	"testing"
	"time"
)

func TestEnvString(t *testing.T) {
	env := &EnvVar{
		Name:  "path",
		Value: "/usr/local/bin",
	}

	expected := "path=/usr/local/bin"
	got := env.String()

	if expected != got {
		t.Errorf("EnvVar.String() fail:\ngot: %s\nexpected: %s\n",
			got, expected)
	}
}

type taskTestCase struct {
	task            Task
	expectedOutput  string
	expectedCommand []string
}

type taskExpansionTestCase struct {
	task            Task
	vars            []Var
	expectedCommand []string
}

var testEcho = taskTestCase{
	task: Task{
		Name: "task1",
		Command: []string{
			"echo",
			"-n",
			"hello",
		},
		ShowOutput: true,
	},
	expectedCommand: []string{
		"echo",
		"-n",
		"hello",
	},
	expectedOutput: "hello",
}

var testEchoInShell = taskTestCase{
	task: Task{
		Name: "task2",
		Command: []string{
			"echo",
			"-n",
			"hello",
		},
		ShowOutput: true,
		Shell:      "bash",
	},
	expectedCommand: []string{
		"/bin/bash",
		"-c",
		"echo -n hello",
	},
	expectedOutput: "hello",
}

var allTests = []taskTestCase{
	testEcho,
	testEchoInShell,
}

func (test *taskTestCase) doTest(t *testing.T) {
	taskInfo, err := test.task.Run()
	if err != nil {
		t.Errorf("Fail to run task: %s\n", err.Error())
	}

	if !reflect.DeepEqual(taskInfo.Args, test.expectedCommand) {
		t.Errorf("Command mismatch:\ngot: %s\nexpected: %s\n",
			taskInfo.Args, test.expectedCommand)
	}

	if taskInfo.StartAt.After(time.Now()) {
		t.Errorf("StartAt is wrong: %v", taskInfo.StartAt)
	}

	id := "not random ID"
	doneChan := make(chan *CmdDoneChan)
	errChan := make(chan *CmdDoneChan)

	taskInfo.WaitPoll(id, doneChan, errChan)

	isErr := false
	var ret *CmdDoneChan

	select {
	case ret = <-doneChan:
		if ret.ID != id {
			t.Errorf("Task finished with wrong ID: %s\n", ret.ID)
		}
	case ret = <-errChan:
		isErr = true
		t.Errorf("Task finished with error: %s\n", ret.Error)
		if ret.ID != id {
			t.Errorf("Task error and wrong id: %s\n", ret.Error)
		}
	}

	if !isErr {
		if test.task.ShowOutput {
			output := ret.Output
			if output != test.expectedOutput {
				t.Errorf("Task output mismatch:\ngot: %s\nexpected: %s\n",
					output, test.expectedOutput)
			}
		}
	}
}

func TestTasks(t *testing.T) {
	for _, testCase := range allTests {
		testCase.doTest(t)
	}
}

var allExpansionTest = []taskExpansionTestCase{
	{
		task: Task{
			Command: []string{
				"tar",
				"-cjf",
				"${current_dir}",
				"${old_dir}",
				"${tar_flags}",
			},
		},
		vars: []Var{
			{
				Name:  "current_dir",
				Value: "/root",
			}, {
				Name:  "old_dir",
				Value: "/tmp",
			}, {
				Name:  "tar_flags",
				Value: "-xfzabc",
			},
		},
		expectedCommand: []string{
			"tar",
			"-cjf",
			"/root",
			"/tmp",
			"-xfzabc",
		},
	},
}

func (test *taskExpansionTestCase) doTest(t *testing.T) {
	test.task.preRunAddVars(test.vars)
	if reflect.DeepEqual(test.task.Command, test.expectedCommand) {
		t.Errorf("Wrong expansion:\ngot: %v\nexpected: %v\n",
			test.task.Command, test.expectedCommand)
	}
}

func TestExpandTask(t *testing.T) {
	for _, testCase := range allExpansionTest {
		testCase.doTest(t)
	}
}
