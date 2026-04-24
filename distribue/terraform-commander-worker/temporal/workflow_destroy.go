package temporal

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/fabienChaillou/terraform-contracts/contracts"
)

// DestroyWorkflow executes `terraform destroy` on the restricted ApplyTaskQueue.
//
// Destroy is NOT idempotent — it MUST NOT be retried automatically.
// MaxAttempts is hard-coded to 1, identical to ApplyWorkflow.
//
// The default timeout is 60 minutes (more generous than apply) because
// destroy operations can block on dependent resources being deleted in the
// correct order.
//
// Worker registration:
//
//	w.RegisterWorkflowWithOptions(DestroyWorkflow, workflow.RegisterOptions{
//	    Name: contracts.WorkflowDestroy,
//	})
func DestroyWorkflow(ctx workflow.Context, input contracts.WorkflowInput) (*contracts.ExecutionResult, error) {
	timeout := time.Duration(input.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 60 * time.Minute
	}

	ao := workflow.ActivityOptions{
		TaskQueue:           contracts.TaskQueue,
		StartToCloseTimeout: timeout,
		HeartbeatTimeout:    180 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			// Destroy is non-idempotent: hard-code MaxAttempts = 1.
			MaximumAttempts: 1,
		},
	}

	workflow.GetLogger(ctx).Info("DestroyWorkflow: starting destroy (no retry)")

	var result *contracts.ExecutionResult
	err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, ao),
		ExecuteShellCommand,
		ActivityInput{Args: input.Args},
	).Get(ctx, &result)

	return result, err
}
