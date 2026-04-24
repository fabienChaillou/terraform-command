package temporal

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/fabienChaillou/terraform-contracts/contracts"
)

// PlanWorkflow executes `terraform plan` on the standard task queue.
//
// Plan is read-only and idempotent — it can be safely retried.
// Exit code 2 means "changes detected" and is a valid, non-error outcome.
//
// Worker registration:
//
//	w.RegisterWorkflowWithOptions(PlanWorkflow, workflow.RegisterOptions{
//	    Name: contracts.WorkflowPlan,
//	})
func PlanWorkflow(ctx workflow.Context, input contracts.WorkflowInput) (*contracts.ExecutionResult, error) {
	timeout := time.Duration(input.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 15 * time.Minute
	}

	maxAttempts := input.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3 // idempotent — safe to retry
	}

	ao := workflow.ActivityOptions{
		TaskQueue:           contracts.TaskQueue,
		StartToCloseTimeout: timeout,
		HeartbeatTimeout:    60 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: maxAttempts,
		},
	}

	var result *contracts.ExecutionResult
	err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, ao),
		ExecuteShellCommand,
		ActivityInput{Args: input.Args},
	).Get(ctx, &result)

	if err != nil {
		return nil, err
	}

	// Exit code 2 = plan succeeded with pending changes (not an error).
	if result != nil && result.ExitCode == 2 {
		workflow.GetLogger(ctx).Info("PlanWorkflow: changes detected (exit code 2)")
	}

	return result, nil
}
