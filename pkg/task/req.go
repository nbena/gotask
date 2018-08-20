package task

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
