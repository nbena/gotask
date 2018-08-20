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
	"syscall"

	"github.com/nbena/gotask/pkg/task"
)

// TaskServer is the HTTP server
type TaskServer struct {
	// server *http.ServerMux
	taskMap        TaskMap
	pendingTasks   longRunningTasksMap
	completedTasks longRunningTasksMap

	completeChan chan string
	errChan      chan [2]string

	config *RuntimeConfig

	taskManagerCloseChan chan os.Signal
	serverCloseChan      chan os.Signal

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

	taskMap := TaskMap{
		tasks: make(map[string]task.Task),
	}

	if err = taskMap.ReadTasks(config.TaskFile, false); err != nil {
		return nil, err
	}

	server := &TaskServer{
		taskMap: taskMap,
		pendingTasks: longRunningTasksMap{
			taskMap: make(map[string]task.RuntimeTaskInfo),
		},
		completedTasks: longRunningTasksMap{
			taskMap: make(map[string]task.RuntimeTaskInfo),
		},
		completeChan: make(chan string, config.InternalChanSize),
		errChan:      make(chan [2]string, config.InternalChanSize),
		config: &RuntimeConfig{
			taskFilePath: config.TaskFile,
			logRequests:  config.LogRequests,
		},
		taskManagerCloseChan: make(chan os.Signal),
		serverCloseChan:      make(chan os.Signal),
		listener:             listener,
		// mux:                  http.NewServeMux(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/refresh", server.refresh)
	mux.HandleFunc("/list", server.list)
	mux.HandleFunc("/exec", server.execute)
	mux.HandleFunc("/poll", server.poll)

	server.httpServer = &http.Server{
		Handler: mux,
	}

	signal.Notify(server.taskManagerCloseChan, syscall.SIGTERM, syscall.SIGSTOP, syscall.SIGINT)
	signal.Notify(server.serverCloseChan, syscall.SIGTERM, syscall.SIGSTOP, syscall.SIGINT)

	return server, nil
}

// Run starts the server.
// it's a blocking call.
func (t *TaskServer) Run() {
	go func() {
		if err := t.httpServer.Serve(t.listener); err != nil {
			log.Printf("Error in listen: %s\n", err.Error())
			t.taskManagerCloseChan <- syscall.SIGTERM
		}
	}()
	go func() {
		t.taskManager()
	}()

	<-t.serverCloseChan
}

type taskID struct {
	task.Task
	ID string
}

func (t *TaskServer) moveTasks(id string, unlockCompleted bool) task.RuntimeTaskInfo {
	// first remove from the pending map
	t.pendingTasks.Lock()
	old := t.pendingTasks.taskMap[id]
	delete(t.pendingTasks.taskMap, id)
	t.pendingTasks.Unlock()

	// the move to the complete map
	t.completedTasks.Lock()
	t.completedTasks.taskMap[id] = old

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
		case id := <-t.completeChan:
			// ok the task is finished, we move it
			// to the completed tasks map
			t.moveTasks(id, true)

		case errDesc := <-t.errChan:

			old := t.moveTasks(errDesc[0], false)
			// we write the error to stderr
			old.Cmd.Stderr.Write([]byte("ERROR IN WAIT: " + errDesc[1]))
		// exit
		case <-t.taskManagerCloseChan:
			loop = false
		}
	}
}
