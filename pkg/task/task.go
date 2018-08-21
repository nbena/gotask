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
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
)

// EnvVar are the env to be used in a Task.
type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"val"`
}

func (e *EnvVar) String() string {
	return fmt.Sprintf("%s=%s", e.Name, e.Value)
}

// Task wraps all info about a task to run.
type Task struct {
	// A unique name
	Name string `json:"name"`

	// the command to run
	Command []string `json:"command"`

	// optional, the directory in which to run it
	Dir string `json:"dir"`

	// does it takes a long?
	Long bool `json:"isLong"`

	// you want the output of the command?
	ShowOutput bool `json:"showOutput"`

	// env
	Env []EnvVar `json:"env"`

	// run using a shell? which one
	// considered only if not empty
	// the option '-c' will be then used
	Shell string `json:"shell"`
}

// RuntimeTaskInfo keeps only the necessary info
// for a long-running task
type RuntimeTaskInfo struct {
	// misuring exec tim
	StartAt time.Time
	EndAt   time.Time
	// show output?
	ShowOutput bool

	// pipe for stdout
	OutPipe io.ReadCloser
	// pipe for stderr
	ErrPipe io.ReadCloser
	*exec.Cmd

	// these are set after the call to wait
	// by the channel receiver because
	// this object is in a table that may be
	// accessed at the same time
	Output string
	Error  string
}

// CmdDoneChan is the struct written on the 'done' channel.
// It is used on 'err' chan too.
type CmdDoneChan struct {
	Output string
	Error  string
	ID     string
}

// WaitPoll waits the command to complete,
// writing its ID on done when finished.
// Any error will be reported to err.
func (r *RuntimeTaskInfo) WaitPoll(
	id string,
	doneChan chan<- *CmdDoneChan,
	errChan chan<- *CmdDoneChan) {

	go func() {
		// we have to read output BEFORE call to Wait()
		outStr, err := internalPipeToStr(r.OutPipe)
		if err != nil {
			errChan <- &CmdDoneChan{
				ID:    id,
				Error: fmt.Sprintf("Fail to get STDOUT: %s\n", err.Error()),
			}
		}

		// read error
		errStr, err := internalPipeToStr(r.ErrPipe)
		if err != nil {
			errChan <- &CmdDoneChan{
				ID:    id,
				Error: fmt.Sprintf("Fail to get STDERR: %s\n", err.Error()),
			}
		}

		// not wait
		if err = r.Wait(); err != nil {
			r.EndAt = time.Now()
			errChan <- &CmdDoneChan{
				ID:    id,
				Error: err.Error(),
			}
		} else {
			r.EndAt = time.Now()
			doneChan <- &CmdDoneChan{
				ID:     id,
				Output: outStr,
				Error:  errStr,
			}
		}
	}()
}

func internalPipeToStr(pipe io.ReadCloser) (string, error) {
	out, err := ioutil.ReadAll(pipe)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// StdoutStr returns the output in string format
// func (r *RuntimeTaskInfo) StdoutStr() (string, error) {
// 	return internalPipeToStr(r.OutPipe)
// }

// StderrStr returns the output in string format
// func (r *RuntimeTaskInfo) StderrStr() (string, error) {
// 	return internalPipeToStr(r.ErrPipe)
// }

func (t *Task) String() string {
	var writer strings.Builder
	encoder := json.NewEncoder(&writer)
	// WE SKIP THE ERROR
	encoder.Encode(t)
	return writer.String()
}

// preRun takes care of adding eventual variables
// TODO add some default variable
func (t *Task) preRunAddVars(vars []Var) {

	for _, variable := range vars {
		for _, command := range t.Command {
			command = strings.Replace(
				command,
				variable.ToReplacer(),
				variable.Value,
				-1)
		}
	}
}

func (t *Task) preRunGetVars() ([]Var, error) {
	return ReadVars(path.Join(t.Dir, VarFileName))
}

func (t *Task) preRun() error {
	if t.Dir != "" {
		vars, err := t.preRunGetVars()
		if err != nil {
			return err
		}
		t.preRunAddVars(vars)
	}
	return nil
}

// Run runs the task in a non-blocking way
// returning the RuntimeTaskInfo associated with.
func (t *Task) Run() (*RuntimeTaskInfo, error) {

	if err := t.preRun(); err != nil {
		return nil, err
	}

	commands := t.Command

	var cmd *exec.Cmd

	if t.Shell != "" {
		baseShell, err := exec.LookPath(t.Shell)
		if err != nil {
			return nil, err
		}
		cmd = exec.Command(baseShell, "-c", strings.Join(commands, " "))
	} else {
		cmd = exec.Command(commands[0], commands[1:]...)
	}

	// now add other info if needed
	if len(t.Env) > 0 {
		envs := make([]string, len(t.Env))
		for i, env := range t.Env {
			envs[i] = env.String()
		}
		cmd.Env = append(os.Environ(), envs...)
	}

	// set the path, if empty it's just fine
	// because the Cmd works the same way
	cmd.Dir = t.Dir

	pipeOut, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	pipeErr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	runtimeTask := &RuntimeTaskInfo{
		Cmd:        cmd,
		ShowOutput: t.ShowOutput,
		OutPipe:    pipeOut,
		ErrPipe:    pipeErr,
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	runtimeTask.StartAt = time.Now()

	return runtimeTask, nil
}
