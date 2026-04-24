package temporal

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/fabienChaillou/terraform-commander/command"
)

// ApplyTaskQueue is a dedicated task queue for apply and destroy operations.
// Running these on a separate queue means:
//   - Only hardened, audited workers can execute destructive commands.
//   - Operators can scale apply-workers independently of plan-workers.
//   - A bug in a plan-worker cannot accidentally trigger an apply.
const ApplyTaskQueue = "terraform-apply-queue"

// ApplyWorkflow is the Temporal workflow for "terraform apply".
//
// It differs from the generic ShellCommandWorkflow in three ways:
//
//  1. Hard-coded MaxAttempts = 1: apply is never retried automatically.
//     An interrupted apply must be investigated by a human before re-running.
//
//  2. Dedicated task queue (ApplyTaskQueue): only workers registered on this
//     queue execute apply commands, providing an audit and access-control
//     boundary.
//
//  3. Longer HeartbeatTimeout (120 s): apply can be slow; we give it more
//     time between heartbeats before Temporal considers the worker lost.
//
// The function signature must match WorkflowInput / *command.ExecutionResult
// so the executor can submit it with the same WorkflowInput as ShellCommandWorkflow.
func ApplyWorkflow(
	ctx workflow.Context,
	input WorkflowInput,
) (*command.ExecutionResult, error) {
	logger := workflow.GetLogger(ctx)
	info := workflow.GetInfo(ctx)

	logger.Info("ApplyWorkflow: starting",
		"workflow_id", info.WorkflowExecution.ID,
		"args", input.Args,
	)

	timeoutSec := input.TimeoutSeconds
	if timeoutSec <= 0 {
		timeoutSec = 1800 // 30 min — apply can take a long time
	}

	ao := workflow.ActivityOptions{
		// Dedicated queue — only apply-qualified workers pick this up.
		TaskQueue: ApplyTaskQueue,

		StartToCloseTimeout: time.Duration(timeoutSec) * time.Second,

		// Apply operations are slow; allow more time between heartbeats.
		HeartbeatTimeout: 120 * time.Second,

		RetryPolicy: &temporal.RetryPolicy{
			// Hard-coded: apply is NOT idempotent — never retry automatically.
			// If the workflow fails, Temporal marks it as failed and the caller
			// receives an error; a human must investigate before retrying.
			MaximumAttempts: 1,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var result *command.ExecutionResult
	err := workflow.ExecuteActivity(ctx, ExecuteShellCommand, ActivityInput{
		Args: input.Args,
	}).Get(ctx, &result)

	if err != nil {
		logger.Error("ApplyWorkflow: activity failed", "error", err)
		return nil, err
	}

	logger.Info("ApplyWorkflow: completed",
		"workflow_id", info.WorkflowExecution.ID,
		"exit_code", result.ExitCode,
		"success", result.ExitCode == 0,
	)
	return result, nil
}
