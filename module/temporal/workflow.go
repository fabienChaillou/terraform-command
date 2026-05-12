package temporal

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/fabienChaillou/terraform-commander/internal/terraform"
)

// TaskQueue is the standard task queue polled by the generic worker.
// init, workspace, and plan use this queue.
const TaskQueue = "shell-command-queue"

// WorkflowInput is the JSON-serialisable input shared by all Terraform workflows.
//
// The binary is absent: it is fixed by the activity's TerraformBinary constant
// and must not be configurable at runtime to prevent arbitrary command injection.
type WorkflowInput struct {
	// Args is the flat CLI argument slice forwarded verbatim to the activity.
	Args []string `json:"args"`

	// TimeoutSeconds is the StartToCloseTimeout for the activity.
	// 0 falls back to DefaultTimeoutSeconds.
	TimeoutSeconds int `json:"timeout_seconds,omitempty"`

	// MaxAttempts is the maximum number of activity execution attempts.
	// 0 or 1 means no retry.
	MaxAttempts int32 `json:"max_attempts,omitempty"`
}

// DefaultTimeoutSeconds is the activity timeout used when WorkflowInput.TimeoutSeconds is 0.
const DefaultTimeoutSeconds = 1800 // 30 minutes

// ShellCommandWorkflow is the generic fallback Temporal workflow.
// It is used for any action that does not have a dedicated workflow registered
// in TemporalExecutor.WorkflowByAction.
//
// All five standard Terraform actions (init, plan, apply, destroy, workspace)
// have dedicated workflows.  ShellCommandWorkflow exists for future or custom
// actions that do not yet have their own workflow.
func ShellCommandWorkflow(
	ctx workflow.Context,
	input WorkflowInput,
) (*terraform.ExecutionResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("ShellCommandWorkflow: started", "args", input.Args)

	timeoutSec := input.TimeoutSeconds
	if timeoutSec <= 0 {
		timeoutSec = DefaultTimeoutSeconds
	}
	maxAttempts := input.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	ao := workflow.ActivityOptions{
		TaskQueue:           TaskQueue,
		StartToCloseTimeout: time.Duration(timeoutSec) * time.Second,
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: maxAttempts},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var result *terraform.ExecutionResult
	err := workflow.ExecuteActivity(ctx, ExecuteShellCommand, ActivityInput{
		Args: input.Args,
	}).Get(ctx, &result)
	if err != nil {
		return nil, err
	}

	logger.Info("ShellCommandWorkflow: completed", "exit_code", result.ExitCode)
	return result, nil
}
