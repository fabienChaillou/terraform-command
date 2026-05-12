package temporal

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/fabienChaillou/terraform-commander/internal/terraform"
)

// InitWorkflow is the Temporal workflow for "terraform init".
//
// Characteristics:
//   - Retryable (up to MaxAttempts, default 3): init is idempotent — it can
//     be safely retried on transient failures (provider registry timeout,
//     plugin download error, worker restart).
//   - Short timeout (300 s default): init should complete quickly; a long
//     duration usually signals a stuck provider download.
//   - Standard queue: init does not require elevated privileges.
func InitWorkflow(
	ctx workflow.Context,
	input WorkflowInput,
) (*terraform.ExecutionResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("InitWorkflow: starting", "args", input.Args)

	timeoutSec := input.TimeoutSeconds
	if timeoutSec <= 0 {
		timeoutSec = 300 // 5 min — init should be fast
	}

	maxAttempts := input.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3 // init is idempotent, safe to retry
	}

	ao := workflow.ActivityOptions{
		TaskQueue:           TaskQueue,
		StartToCloseTimeout: time.Duration(timeoutSec) * time.Second,
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: maxAttempts,
			// Only OS-level errors are retried; non-zero exit codes are
			// returned as results and never trigger a retry.
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var result *terraform.ExecutionResult
	err := workflow.ExecuteActivity(ctx, ExecuteShellCommand, ActivityInput{
		Args: input.Args,
	}).Get(ctx, &result)

	if err != nil {
		logger.Error("InitWorkflow: activity failed", "error", err)
		return nil, err
	}

	logger.Info("InitWorkflow: completed", "exit_code", result.ExitCode)
	return result, nil
}
