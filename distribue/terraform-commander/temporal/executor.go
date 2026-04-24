// Package temporal implements the api.Executor interface by submitting a
// Temporal workflow and waiting synchronously for the result.
//
// # String-based workflow routing
//
// The server never imports the worker repo.  Instead, it calls workflows by
// NAME (a string constant from terraform-contracts), not by function
// reference.  The worker registers its workflow functions under those exact
// name strings via workflow.RegisterOptions.
//
// This zero-coupling approach is the key architectural decision behind the
// two-repo split: adding or changing a workflow only requires updating the
// worker repo and the contracts repo — the server is untouched.
package temporal

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/client"

	"github.com/fabienChaillou/terraform-commander/command"
	"github.com/fabienChaillou/terraform-contracts/contracts"
)

// WorkflowStarter is the narrow subset of client.Client required by
// TemporalExecutor.  Using an interface (rather than the full client.Client)
// allows the executor to be unit-tested with a lightweight mock — no real
// Temporal server is needed.
//
// *client.Client satisfies this interface via structural typing, so callers
// that pass a real Temporal client require no changes.
type WorkflowStarter interface {
	ExecuteWorkflow(
		ctx context.Context,
		options client.StartWorkflowOptions,
		workflow interface{},
		args ...interface{},
	) (client.WorkflowRun, error)
}

// TemporalExecutor implements api.Executor by dispatching a named Temporal
// workflow and blocking until it completes.
//
// # Workflow routing
//
// WorkflowByAction maps action names to workflow NAME STRINGS (from
// contracts).  Absent actions fall back to contracts.WorkflowShellCommand.
// All workflows are dispatched to contracts.TaskQueue — a single worker
// handles every action.
//
// # Example
//
//	executor := &temporal.TemporalExecutor{
//	    Client: temporalClient,
//	    DefaultTimeout: 30 * time.Minute,
//	    MaxAttempts:    1,
//	    WorkflowByAction: contracts.ActionMap[string]{
//	        "init":      contracts.WorkflowInit,
//	        "plan":      contracts.WorkflowPlan,
//	        "apply":     contracts.WorkflowApply,
//	        "destroy":   contracts.WorkflowDestroy,
//	        "workspace": contracts.WorkflowWorkspace,
//	    },
//	}
type TemporalExecutor struct {
	// Client is the Temporal workflow starter.  Pass a real *client.Client in
	// production; inject a mockWorkflowStarter in tests.
	Client WorkflowStarter

	// DefaultTimeout is the fallback StartToCloseTimeout when ExecuteOptions
	// does not carry an explicit timeout.
	DefaultTimeout time.Duration

	// MaxAttempts is the default maximum number of activity attempts.
	// 1 = no retry (recommended for apply / destroy).
	MaxAttempts int32

	// WorkflowByAction maps an action name to the workflow NAME to invoke.
	// Values must match the names used in worker.RegisterWorkflowWithOptions.
	WorkflowByAction contracts.ActionMap[string]
}

// resolveWorkflow returns the workflow name for the given action.
// Falls back to WorkflowShellCommand for unknown/empty actions.
func (e *TemporalExecutor) resolveWorkflow(action string) string {
	return e.WorkflowByAction.Get(action, contracts.WorkflowShellCommand)
}

// Execute submits the named workflow for args and blocks until the workflow
// completes, returning the process exit code, stdout, and stderr.
//
// Implements api.Executor:
//
//	Execute(ctx, args, opts) (*command.ExecutionResult, error)
//
// The terraform binary is intentionally absent from this signature — it is a
// compile-time constant fixed in the worker's activity package
// (TerraformBinary), not configurable by the server.
func (e *TemporalExecutor) Execute(
	ctx context.Context,
	args []string,
	opts command.ExecuteOptions,
) (*command.ExecutionResult, error) {
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

	// ── Select workflow name ──────────────────────────────────────────────────
	workflowName := e.resolveWorkflow(opts.Action)
	workflowID := fmt.Sprintf("tf-%s-%d", opts.Action, time.Now().UnixNano())

	// ── Submit workflow ───────────────────────────────────────────────────────
	we, err := e.Client.ExecuteWorkflow(ctx,
		client.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: contracts.TaskQueue,
		},
		workflowName, // string, not a function reference — zero import coupling
		contracts.WorkflowInput{
			Args:           args,
			TimeoutSeconds: timeoutSec,
			MaxAttempts:    maxAttempts,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("temporal: start workflow %s: %w", workflowID, err)
	}

	// ── Wait for result ───────────────────────────────────────────────────────
	var result *command.ExecutionResult
	if err := we.Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("temporal: workflow %s failed: %w", workflowID, err)
	}

	return result, nil
}
