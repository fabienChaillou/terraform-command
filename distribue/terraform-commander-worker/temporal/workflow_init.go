package temporal

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/fabienChaillou/terraform-contracts/contracts"
)

// InitWorkflow executes `terraform init` on the standard task queue.
//
// Init is idempotent — it can be safely retried.  The default timeout is
// 5 minutes, which is generous for plugin downloads on cold caches.
//
// Worker registration:
//
//	w.RegisterWorkflowWithOptions(InitWorkflow, workflow.RegisterOptions{
//	    Name: contracts.WorkflowInit,
//	})
func InitWorkflow(ctx workflow.Context, input contracts.WorkflowInput) (*contracts.ExecutionResult, error) {
	timeout := time.Duration(input.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}

	maxAttempts := input.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3 // idempotent — safe to retry
	}

	ao := workflow.ActivityOptions{
		TaskQueue:           contracts.TaskQueue,
		StartToCloseTimeout: timeout,
		HeartbeatTimeout:    30 * time.Second,
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

	return result, err
}
