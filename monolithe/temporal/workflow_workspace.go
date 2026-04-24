package temporal

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/fabienChaillou/terraform-commander/command"
)

// WorkspaceWorkflow is the Temporal workflow for "terraform workspace" subcommands
// (new | select | list | delete | show).
//
// Characteristics:
//   - Retryable (up to MaxAttempts, default 2): workspace operations are
//     metadata-only and generally safe to retry on transient failures.
//   - Very short timeout (60 s default): workspace commands interact only with
//     the state backend and complete in seconds; a long duration signals a
//     connectivity issue with the backend.
//   - Short HeartbeatTimeout (15 s): allows Temporal to detect a stuck worker
//     quickly for these fast-running operations.
//   - Standard queue: workspace operations do not require elevated privileges.
func WorkspaceWorkflow(
	ctx workflow.Context,
	input WorkflowInput,
) (*command.ExecutionResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("WorkspaceWorkflow: starting", "args", input.Args)

	timeoutSec := input.TimeoutSeconds
	if timeoutSec <= 0 {
		timeoutSec = 60 // 1 min — workspace commands are near-instant
	}

	maxAttempts := input.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 2 // workspace is idempotent for list/show/select
	}

	ao := workflow.ActivityOptions{
		TaskQueue:           TaskQueue,
		StartToCloseTimeout: time.Duration(timeoutSec) * time.Second,
		HeartbeatTimeout:    15 * time.Second, // fast ops → detect stuck workers quickly
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: maxAttempts,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var result *command.ExecutionResult
	err := workflow.ExecuteActivity(ctx, ExecuteShellCommand, ActivityInput{
		Args: input.Args,
	}).Get(ctx, &result)

	if err != nil {
		logger.Error("WorkspaceWorkflow: activity failed", "error", err)
		return nil, err
	}

	logger.Info("WorkspaceWorkflow: completed", "exit_code", result.ExitCode)
	return result, nil
}
