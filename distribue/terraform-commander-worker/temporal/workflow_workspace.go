package temporal

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/fabienChaillou/terraform-contracts/contracts"
)

// WorkspaceWorkflow executes `terraform workspace <subcommand>` on the standard
// task queue.
//
// Workspace operations (new, select, list, delete, show) are fast metadata
// operations — the default timeout is 1 minute.  New and delete can be safely
// retried if the worker crashes mid-flight; select/list/show are naturally
// idempotent.
//
// Worker registration:
//
//	w.RegisterWorkflowWithOptions(WorkspaceWorkflow, workflow.RegisterOptions{
//	    Name: contracts.WorkflowWorkspace,
//	})
func WorkspaceWorkflow(ctx workflow.Context, input contracts.WorkflowInput) (*contracts.ExecutionResult, error) {
	timeout := time.Duration(input.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 1 * time.Minute
	}

	maxAttempts := input.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 2
	}

	ao := workflow.ActivityOptions{
		TaskQueue:           contracts.TaskQueue,
		StartToCloseTimeout: timeout,
		HeartbeatTimeout:    15 * time.Second,
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
