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
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/nbena/gotask/pkg/task"
)

const (
	MethodList    = http.MethodGet
	MethodRefresh = http.MethodPost
	MethodExecute = http.MethodPut
	MethodPoll    = http.MethodGet
	MethodAdd     = http.MethodPost

	StatusList    = http.StatusOK
	StatusRefresh = http.StatusNoContent
	StatusExecute = http.StatusOK
	StatusPoll    = http.StatusOK
	StatusAdd     = http.StatusNoContent

	StatusAddConflict = http.StatusConflict
	// StatusNotFound    = http.StatusNotFound
)

// TaskServer is the HTTP server
type TaskServer struct {
	// server *http.ServerMux
	taskMap        taskMap
	pendingTasks   longRunningTasksMap
	completedTasks longRunningTasksMap

	taskDoneChan chan *task.CmdDoneChan
	taskErrChan  chan *task.CmdDoneChan

	config *RuntimeConfig

	taskManagerCloseChan chan os.Signal
	ServerCloseChan      chan os.Signal

	listener net.Listener

	// mux *http.ServeMux
	httpServer *http.Server
}

// NewServer returns a new server instance.
func NewServer(config *Config) (*TaskServer, error) {

	listener, err := net.Listen("tcp4", fmt.Sprintf("%s:%d",
		config.ListenAddr, config.ListenPort))
	if err != nil {
		return nil, err
	}

	// taskMap := TaskMap{
	// 	tasks:   make(map[string]task.Task),
	// 	RWMutex: &sync.RWMutex{},
	// }
	taskMap := taskMap{
		Map: &sync.Map{},
	}

	if err = taskMap.ReadTasks(config.TaskFile, false); err != nil {
		return nil, err
	}

	server := &TaskServer{
		taskMap: taskMap,
		pendingTasks: longRunningTasksMap{
			taskMap: make(map[string]task.RuntimeTaskInfo),
			RWMutex: &sync.RWMutex{},
		},
		completedTasks: longRunningTasksMap{
			taskMap: make(map[string]task.RuntimeTaskInfo),
			RWMutex: &sync.RWMutex{},
		},
		taskDoneChan: make(chan *task.CmdDoneChan, config.InternalChanSize),
		taskErrChan:  make(chan *task.CmdDoneChan, config.InternalChanSize),
		config: &RuntimeConfig{
			taskFilePath: config.TaskFile,
			logRequests:  config.LogRequests,
		},
		taskManagerCloseChan: make(chan os.Signal),
		ServerCloseChan:      make(chan os.Signal),
		listener:             listener,
		// mux:                  http.NewServeMux(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/refresh", server.refresh)
	mux.HandleFunc("/list", server.list)
	mux.HandleFunc("/exec", server.execute)
	mux.HandleFunc("/poll", server.poll)
	mux.HandleFunc("/add", server.add)

	server.httpServer = &http.Server{
		Handler: mux,
	}

	signal.Notify(server.taskManagerCloseChan, syscall.SIGTERM, syscall.SIGSTOP, syscall.SIGINT)
	signal.Notify(server.ServerCloseChan, syscall.SIGTERM, syscall.SIGSTOP, syscall.SIGINT)

	return server, nil
}

// Run starts the server.
// it's a blocking call.
func (t *TaskServer) Run() {
	go func() {
		if err := t.httpServer.Serve(t.listener); err != nil {
			log.Printf("Error in listen: %s\n", err.Error())
			t.taskManagerCloseChan <- syscall.SIGTERM
			t.ServerCloseChan <- syscall.SIGTERM
		}
	}()
	go func() {
		t.taskManager()
	}()

	<-t.ServerCloseChan
	t.taskManagerCloseChan <- syscall.SIGTERM
	t.httpServer.Close()
	t.listener.Close()
}

// type taskID struct {
// 	task.Task
// 	ID string
// }

func (t *TaskServer) moveTasks(res *task.CmdDoneChan, unlockCompleted bool) task.RuntimeTaskInfo {
	// first remove from the pending map
	t.pendingTasks.Lock()
	old := t.pendingTasks.taskMap[res.ID]
	delete(t.pendingTasks.taskMap, res.ID)
	t.pendingTasks.Unlock()

	// passing the results
	old.Output = res.Output
	old.Error = res.Error

	// the move to the complete map
	t.completedTasks.Lock()
	t.completedTasks.taskMap[res.ID] = old

	// the called may need to do other operation before unlock
	if unlockCompleted {
		t.completedTasks.Unlock()
	}

	return old
}

// this manages the different running processes
func (t *TaskServer) taskManager() {
	// TODO shutdown
	loop := true
	for loop {
		select {
		case res := <-t.taskDoneChan:
			// ok the task is finished, we move it
			// to the completed tasks map
			t.moveTasks(res, true)

		case res := <-t.taskErrChan:

			t.moveTasks(res, true)
			// we write the error to stderr
			// old.Cmd.Stderr.Write([]byte("ERROR IN WAIT: " + errDesc[1]))
		// exit
		case <-t.taskManagerCloseChan:
			loop = false
		}
	}
}
