package temporal

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/client"

	"github.com/fabienChaillou/terraform-commander/internal/terraform"
)

// TemporalExecutor implements the api.Executor interface by submitting a
// Temporal workflow and waiting synchronously for the result.
//
// # Workflow routing
//
// Every action is routed to a dedicated workflow via WorkflowByAction.
// Actions absent from the map fall back to ShellCommandWorkflow.
//
//	executor := &TemporalExecutor{
//	    Client: temporalClient,
//	    WorkflowByAction: terraform.ActionMap[interface{}]{
//	        "init":      InitWorkflow,
//	        "plan":      PlanWorkflow,
//	        "apply":     ApplyWorkflow,
//	        "destroy":   DestroyWorkflow,
//	        "workspace": WorkspaceWorkflow,
//	    },
//	}
//
// # Zero circular dependency
//
// Because Go uses structural typing, this struct satisfies api.Executor
// without needing to import the api package.
type TemporalExecutor struct {
	Client client.Client

	// DefaultTimeout is the fallback StartToCloseTimeout when ExecuteOptions
	// does not carry an explicit timeout.
	DefaultTimeout time.Duration

	// MaxAttempts is the default maximum number of activity attempts.
	// 1 = no retry (recommended for apply / destroy).
	MaxAttempts int32

	// WorkflowByAction maps a terraform action name to the Temporal workflow
	// function that should handle it.
	//
	// The value type is interface{} because Temporal's client.ExecuteWorkflow
	// accepts any workflow function — there is no common Go interface for it.
	// Absent actions fall back to ShellCommandWorkflow (see resolveWorkflow).
	WorkflowByAction terraform.ActionMap[interface{}]
}

// resolveWorkflow returns the workflow function for the given action.
// Falls back to ShellCommandWorkflow for unknown or empty actions.
func (e *TemporalExecutor) resolveWorkflow(action string) interface{} {
	return e.WorkflowByAction.Get(action, interface{}(ShellCommandWorkflow))
}

// Execute submits the appropriate workflow for args and blocks until the
// workflow completes, returning the process's exit code, stdout, and stderr.
//
// Signature matches api.Executor:
//
//	Execute(ctx, args, opts) (*terraform.ExecutionResult, error)
//
// The binary is intentionally absent from this signature — it is fixed by
// the TerraformBinary constant in the activity package.
func (e *TemporalExecutor) Execute(
	ctx context.Context,
	args []string,
	opts terraform.ExecuteOptions,
) (*terraform.ExecutionResult, error) {
	// ── Resolve timeout ───────────────────────────────────────────────────────
	timeoutSec := opts.TimeoutSeconds
	if timeoutSec <= 0 && e.DefaultTimeout > 0 {
		timeoutSec = int(e.DefaultTimeout.Seconds())
	}

	// ── Resolve max attempts ──────────────────────────────────────────────────
	maxAttempts := opts.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = e.MaxAttempts
	}
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	// ── Select workflow ───────────────────────────────────────────────────────
	workflowFn := e.resolveWorkflow(opts.Action)
	workflowID := fmt.Sprintf("tf-%s-%d", opts.Action, time.Now().UnixNano())

	we, err := e.Client.ExecuteWorkflow(ctx,
		client.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: TaskQueue,
		},
		workflowFn,
		WorkflowInput{
			Args:           args,
			TimeoutSeconds: timeoutSec,
			MaxAttempts:    maxAttempts,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("temporal: start workflow %s: %w", workflowID, err)
	}

	var result *terraform.ExecutionResult
	if err := we.Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("temporal: workflow %s failed: %w", workflowID, err)
	}

	return result, nil
}
