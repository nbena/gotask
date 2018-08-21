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
	"os"

	"github.com/nbena/gotask/pkg/task"
)

type serverTestCase struct {
	tasks  []task.Task
	config *Config
}

// type serverTestCase struct {
// 	server *TaskServer
// }

func (s *serverTestCase) initServer() (*TaskServer, error) {

	// we write tasks and config to file
	taskFile, err := os.Create(s.config.TaskFile)
	if err != nil {
		return nil, err
	}

	encoder := json.NewEncoder(taskFile)
	if encoder.Encode(s.tasks) != nil {
		return nil, err
	}

	taskFile.Close()

	server, err := NewServer(s.config)
	if err != nil {
		return nil, err
	}
}
