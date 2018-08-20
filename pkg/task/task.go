package task

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// EnvVar are the env to be used in a Task.
type EnvVar struct {
	Key, Value string
}

func (e *EnvVar) String() string {
	return fmt.Sprintf("%s=%s", e.Key, e.Value)
}

// Task wraps all info about a task to run.
type Task struct {
	// A unique name
	Name string

	// the command to run
	// it will be splitted when about to run
	Command string

	// optional, the directory in which to run it
	Dir string

	// does it takes a long?
	Long bool

	// you want the output of the command?
	ShowOutput bool

	// env
	Env []EnvVar
}

// RuntimeTaskInfo keeps only the necessary info
// for a long-running task
type RuntimeTaskInfo struct {
	StartAt, EndAt time.Time
	*exec.Cmd
}

// WaitPoll waits the command to complete,
// writing its ID on done when finished.
// Any error will be reported to err.
func (r *RuntimeTaskInfo) WaitPoll(
	id string,
	done chan<- string,
	err chan [2]string) {

	go func() {
		if waitErr := r.Wait(); waitErr != nil {
			err <- [2]string{waitErr.Error(), id}
		} else {
			done <- id
		}
	}()
}

func (t *Task) String() string {
	var writer strings.Builder
	encoder := json.NewEncoder(&writer)
	// WE SKIP THE ERROR
	encoder.Encode(t)
	return writer.String()
}

// Run runs the task in a non-blocking way
// returning the RuntimeTaskInfo associated with.
func (t *Task) Run() (*RuntimeTaskInfo, error) {
	commands := strings.Split(t.Command, " ")

	// TODO add var parsing

	cmd := exec.Command(commands[0], commands[1:]...)

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

	runtimeTask := &RuntimeTaskInfo{
		Cmd: cmd,
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	runtimeTask.StartAt = time.Now()

	return runtimeTask, nil
}
