package temporal

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/fabienChaillou/terraform-commander/internal/terraform"
)

// PlanWorkflow is the Temporal workflow for "terraform plan".
//
// It differs from the generic ShellCommandWorkflow in two ways:
//
//  1. Retryable (up to MaxAttempts): plan is read-only and idempotent, so
//     transient OS failures (network blip fetching providers, worker restart)
//     can be safely retried.
//
//  2. Non-retryable error types: if terraform itself returns a non-zero exit
//     code it is a result, not an error (see ExecuteShellCommand), so the
//     retry policy only applies to infrastructure-level Go errors.
//
// The output (plan file path, diff summary) is returned via ExecutionResult.
// A downstream "apply" step can read the plan file from shared storage and
// submit an ApplyWorkflow with the "-out" flag pointing to it.
func PlanWorkflow(
	ctx workflow.Context,
	input WorkflowInput,
) (*terraform.ExecutionResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("PlanWorkflow: starting", "args", input.Args)

	timeoutSec := input.TimeoutSeconds
	if timeoutSec <= 0 {
		timeoutSec = 900 // 15 min default for plan
	}

	maxAttempts := input.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3 // plan is idempotent — safe to retry
	}

	ao := workflow.ActivityOptions{
		// Plan runs on the standard queue — no elevated privilege needed.
		TaskQueue:           TaskQueue,
		StartToCloseTimeout: time.Duration(timeoutSec) * time.Second,
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: maxAttempts,
			// Only retry OS-level errors (binary not found, context cancelled).
			// Non-zero exit codes are returned as results, never as errors.
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var result *terraform.ExecutionResult
	err := workflow.ExecuteActivity(ctx, ExecuteShellCommand, ActivityInput{
		Args: input.Args,
	}).Get(ctx, &result)

	if err != nil {
		logger.Error("PlanWorkflow: activity failed", "error", err)
		return nil, err
	}

	logger.Info("PlanWorkflow: completed",
		"exit_code", result.ExitCode,
		"has_changes", result.ExitCode == 2, // terraform plan exits 2 when there are changes
	)
	return result, nil
}
