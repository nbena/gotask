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
	"os"
	"reflect"
	"testing"

	"github.com/nbena/gotask/pkg/server"
	"github.com/nbena/gotask/pkg/task"
)

var tasks = []task.Task{
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
}

var taskToAdd = task.Task{
	Name: "task5",
	Command: []string{
		"echo",
		"go",
	},
	ShowOutput: true,
	Long:       false,
	Shell:      "bash",
}

func end(config *server.Config, t *testing.T) {
	if err := os.Remove(config.TaskFile); err != nil {
		t.Errorf("Error in deleting task file: %s\n", config.TaskFile)
	}
}

func tasksCheck(expected, got []task.Task, t *testing.T) {
	count := 0
	for _, task := range expected {
		for _, gotTask := range got {
			if reflect.DeepEqual(task, gotTask) {
				count++
			}
		}
	}
	if count != len(expected) {
		t.Errorf("/list failed:\ngot: %v\nexpected: %v\n", got, expected)
	}
}

type taskTest interface {
	runTest(t *testing.T)
}

type serverTest struct {
	tests []serverTestCase
}
type clientTest struct {
	tests []clientTestCase
}
