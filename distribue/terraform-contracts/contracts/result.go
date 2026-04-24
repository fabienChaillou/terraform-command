// Package contracts defines the shared types exchanged between the
// terraform-commander (server) and terraform-commander-worker (worker) repos.
//
// It has zero external dependencies so it can be imported by both repos
// without pulling in framework code (Temporal, Huma, Chi …).
package contracts

// ExecutionResult is the value returned by a completed Terraform workflow.
// A non-zero ExitCode indicates a Terraform-level error; the workflow itself
// succeeded in that case (the activity ran to completion).
type ExecutionResult struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

// Success reports whether Terraform exited cleanly (exit code 0).
func (r *ExecutionResult) Success() bool { return r.ExitCode == 0 }

// ExecuteOptions carries execution tuning parameters from the API layer down
// to the Temporal executor.  The Action field is used to route the request to
// the correct workflow and task queue.
type ExecuteOptions struct {
	// Action is the terraform sub-command name ("init", "plan", "apply", …).
	// The executor uses this to look up the right workflow name and task queue.
	Action string

	// TimeoutSeconds is the maximum wall-clock time (in seconds) allowed for
	// the activity.  0 means "use the executor's DefaultTimeout".
	TimeoutSeconds int

	// MaxAttempts is the maximum number of activity retries.
	// 1 = no retry.  0 means "use the executor's default".
	MaxAttempts int32
}
