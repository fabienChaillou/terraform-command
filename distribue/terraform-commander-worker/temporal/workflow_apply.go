package temporal

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/fabienChaillou/terraform-contracts/contracts"
)

// ApplyWorkflow executes `terraform apply` on the restricted ApplyTaskQueue.
//
// Apply is NOT idempotent — it MUST NOT be retried automatically.
// MaxAttempts is hard-coded to 1 in this workflow and cannot be overridden
// by the caller, providing a safety gate against accidental double-applies.
//
// This workflow runs exclusively on workers polling ApplyTaskQueue, which in
// production are isolated on hardened hosts with stricter IAM and network
// policies and concurrency = 1.
//
// Worker registration:
//
//	w.RegisterWorkflowWithOptions(ApplyWorkflow, workflow.RegisterOptions{
//	    Name: contracts.WorkflowApply,
//	})
func ApplyWorkflow(ctx workflow.Context, input contracts.WorkflowInput) (*contracts.ExecutionResult, error) {
	timeout := time.Duration(input.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Minute
	}

	ao := workflow.ActivityOptions{
		TaskQueue:           contracts.TaskQueue,
		StartToCloseTimeout: timeout,
		HeartbeatTimeout:    120 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			// Apply is non-idempotent: hard-code MaxAttempts = 1.
			// Callers cannot override this — the safety guarantee lives here,
			// not in the HTTP layer.
			MaximumAttempts: 1,
		},
	}

	workflow.GetLogger(ctx).Info("ApplyWorkflow: starting apply (no retry)")

	var result *contracts.ExecutionResult
	err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, ao),
		ExecuteShellCommand,
		ActivityInput{Args: input.Args},
	).Get(ctx, &result)

	return result, err
}
