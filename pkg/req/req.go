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

package req

// here definition of Requests/Responses.

// ExecuteMessageRequest represents a /POST request execution
type ExecuteMessageRequest struct {
	TaskName string
}

// ShortRunningTaskResponse is returned after issuing a request
// for a short-running task.
type ShortRunningTaskResponse struct {
	Command string
	Output  string
	Error   string
}

// LongRunningTaskResponse is returned after issuing a request
// for a long-running task.
type LongRunningTaskResponse struct {
	Command string
	ID      string
}

// ErrorMessageResponse is returned upon an error.
type ErrorMessageResponse struct {
	Error string
}

const (
	// PollStatusInProgress defines
	// a task not finished yet.
	PollStatusInProgress = "In Progress"
	// PollStatusCompleted defines a completed task.
	PollStatusCompleted = "Completed"
)

// PollStatusInProgressResponse is returned when you poll
// for a not-finished task.
type PollStatusInProgressResponse struct {
	ID     string
	Status string
}

// PollStatusCompletedResponse is returned when you poll
// for a finished task.
type PollStatusCompletedResponse struct {
	PollStatusInProgressResponse
	ShortRunningTaskResponse
}