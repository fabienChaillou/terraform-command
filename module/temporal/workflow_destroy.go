package temporal

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/fabienChaillou/terraform-commander/internal/terraform"
)

// DestroyWorkflow is the Temporal workflow for "terraform destroy".
//
// It uses the same ApplyTaskQueue as ApplyWorkflow because destroy is
// equally destructive: dedicated, audited workers with restricted access
// should be the only ones executing it.
//
// Key constraints:
//   - MaxAttempts = 1 (hard-coded, never overridable): destroying infrastructure
//     twice is catastrophic — a failed destroy must be manually inspected.
//   - Longest HeartbeatTimeout (180 s): destroying large stacks can be slow.
//   - Dedicated ApplyTaskQueue: same access-control boundary as apply.
func DestroyWorkflow(
	ctx workflow.Context,
	input WorkflowInput,
) (*terraform.ExecutionResult, error) {
	logger := workflow.GetLogger(ctx)
	info := workflow.GetInfo(ctx)

	logger.Info("DestroyWorkflow: starting",
		"workflow_id", info.WorkflowExecution.ID,
		"args", input.Args,
	)

	timeoutSec := input.TimeoutSeconds
	if timeoutSec <= 0 {
		timeoutSec = 3600 // 60 min — destroy can be very slow on large stacks
	}

	ao := workflow.ActivityOptions{
		// Same restricted queue as apply.
		TaskQueue: ApplyTaskQueue,

		StartToCloseTimeout: time.Duration(timeoutSec) * time.Second,

		// Destroy can be very slow; long heartbeat timeout prevents spurious failures.
		HeartbeatTimeout: 180 * time.Second,

		RetryPolicy: &temporal.RetryPolicy{
			// destroy is NOT idempotent and not safe to retry.
			MaximumAttempts: 1,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var result *terraform.ExecutionResult
	err := workflow.ExecuteActivity(ctx, ExecuteShellCommand, ActivityInput{
		Args: input.Args,
	}).Get(ctx, &result)

	if err != nil {
		logger.Error("DestroyWorkflow: activity failed", "error", err)
		return nil, err
	}

	logger.Info("DestroyWorkflow: completed",
		"workflow_id", info.WorkflowExecution.ID,
		"exit_code", result.ExitCode,
		"success", result.ExitCode == 0,
	)
	return result, nil
}
