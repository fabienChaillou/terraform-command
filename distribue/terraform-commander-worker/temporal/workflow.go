package temporal

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/fabienChaillou/terraform-contracts/contracts"
)

// ShellCommandWorkflow is the generic fallback workflow for any Terraform
// action that does not have a dedicated workflow.
//
// In practice the executor maps all five standard actions (init, plan, apply,
// destroy, workspace) to their dedicated workflows.  This workflow exists for
// custom or future actions and as a safety net.
//
// Worker registration (in cmd/worker/main.go):
//
//	w.RegisterWorkflowWithOptions(ShellCommandWorkflow, workflow.RegisterOptions{
//	    Name: contracts.WorkflowShellCommand,
//	})
func ShellCommandWorkflow(ctx workflow.Context, input contracts.WorkflowInput) (*contracts.ExecutionResult, error) {
	timeout := time.Duration(input.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Minute
	}

	maxAttempts := input.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 1
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

	return result, err
}
