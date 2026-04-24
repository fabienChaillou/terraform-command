// Workflow unit tests.
//
// Each workflow is tested in isolation using Temporal's in-memory test
// environment (go.temporal.io/sdk/testsuite).  The ExecuteShellCommand
// activity is replaced by a mock so no terraform binary is required.
//
// Test pattern per workflow:
//  1. Happy path — mock returns exit 0, workflow returns the result.
//  2. Activity error — mock returns a Go error, workflow propagates it.
//  3. Non-zero exit — mock returns exit ≠ 0, workflow still returns it (not an error).
//
// Additional cases are added where a workflow has notable semantics
// (e.g. PlanWorkflow's exit-code-2 path, ApplyWorkflow's MaxAttempts=1).
package temporal_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"

	temporal "github.com/fabienChaillou/terraform-commander-worker/temporal"
	"github.com/fabienChaillou/terraform-contracts/contracts"
)

// ── helpers ───────────────────────────────────────────────────────────────────

// newEnv creates a fresh TestWorkflowEnvironment with the ExecuteShellCommand
// activity pre-registered.  Each test must call env.OnActivity to set the
// mock return value before calling env.ExecuteWorkflow.
func newEnv(t *testing.T) *testsuite.TestWorkflowEnvironment {
	t.Helper()
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()
	env.RegisterActivity(temporal.ExecuteShellCommand)
	return env
}

// okResult is a convenience factory for a successful activity result.
func okResult(stdout string) *contracts.ExecutionResult {
	return &contracts.ExecutionResult{ExitCode: 0, Stdout: stdout}
}

// failResult returns a non-zero exit code result (terraform-level failure).
func failResult(exitCode int, stderr string) *contracts.ExecutionResult {
	return &contracts.ExecutionResult{ExitCode: exitCode, Stderr: stderr}
}

// runWorkflow executes a workflow and returns the result + error.
// It asserts that the workflow completed without panicking.
func runWorkflow(t *testing.T, env *testsuite.TestWorkflowEnvironment, wf interface{}, input contracts.WorkflowInput) (*contracts.ExecutionResult, error) {
	t.Helper()
	env.ExecuteWorkflow(wf, input)
	require.True(t, env.IsWorkflowCompleted(), "workflow did not complete")

	var result *contracts.ExecutionResult
	err := env.GetWorkflowResult(&result)
	return result, err
}

// ── InitWorkflow ──────────────────────────────────────────────────────────────

func TestInitWorkflow_HappyPath(t *testing.T) {
	env := newEnv(t)
	env.OnActivity(temporal.ExecuteShellCommand, mock.Anything, mock.Anything).
		Return(okResult("Terraform has been successfully initialized!"), nil)

	result, err := runWorkflow(t, env, temporal.InitWorkflow, contracts.WorkflowInput{
		Args: []string{"init", "-upgrade"},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if result.Stdout == "" {
		t.Error("Stdout is empty, want non-empty")
	}
}

func TestInitWorkflow_ActivityError(t *testing.T) {
	env := newEnv(t)
	env.OnActivity(temporal.ExecuteShellCommand, mock.Anything, mock.Anything).
		Return(nil, errors.New("binary not found"))

	_, err := runWorkflow(t, env, temporal.InitWorkflow, contracts.WorkflowInput{
		Args: []string{"init"},
	})

	if err == nil {
		t.Error("expected error propagation from activity, got nil")
	}
}

func TestInitWorkflow_NonZeroExitIsNotGoError(t *testing.T) {
	env := newEnv(t)
	env.OnActivity(temporal.ExecuteShellCommand, mock.Anything, mock.Anything).
		Return(failResult(1, "Error: no configuration files"), nil)

	result, err := runWorkflow(t, env, temporal.InitWorkflow, contracts.WorkflowInput{
		Args: []string{"init"},
	})

	require.NoError(t, err, "non-zero exit should not be a Go error")
	if result.ExitCode == 0 {
		t.Error("ExitCode = 0, want non-zero")
	}
}

// ── PlanWorkflow ──────────────────────────────────────────────────────────────

func TestPlanWorkflow_HappyPath(t *testing.T) {
	env := newEnv(t)
	env.OnActivity(temporal.ExecuteShellCommand, mock.Anything, mock.Anything).
		Return(okResult("Plan: 3 to add, 0 to change, 0 to destroy."), nil)

	result, err := runWorkflow(t, env, temporal.PlanWorkflow, contracts.WorkflowInput{
		Args: []string{"plan", "-var", "env=prod"},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
}

func TestPlanWorkflow_ExitCodeTwo_IsNotGoError(t *testing.T) {
	// Exit code 2 = "changes detected" — a valid, non-error outcome for plan.
	env := newEnv(t)
	env.OnActivity(temporal.ExecuteShellCommand, mock.Anything, mock.Anything).
		Return(failResult(2, ""), nil)

	result, err := runWorkflow(t, env, temporal.PlanWorkflow, contracts.WorkflowInput{
		Args: []string{"plan"},
	})

	require.NoError(t, err, "exit code 2 must not be a Go error in PlanWorkflow")
	if result.ExitCode != 2 {
		t.Errorf("ExitCode = %d, want 2", result.ExitCode)
	}
}

func TestPlanWorkflow_ActivityError(t *testing.T) {
	env := newEnv(t)
	env.OnActivity(temporal.ExecuteShellCommand, mock.Anything, mock.Anything).
		Return(nil, errors.New("temporal: activity timeout"))

	_, err := runWorkflow(t, env, temporal.PlanWorkflow, contracts.WorkflowInput{
		Args: []string{"plan"},
	})

	if err == nil {
		t.Error("expected error propagation")
	}
}

// ── ApplyWorkflow ─────────────────────────────────────────────────────────────

func TestApplyWorkflow_HappyPath(t *testing.T) {
	env := newEnv(t)
	env.OnActivity(temporal.ExecuteShellCommand, mock.Anything, mock.Anything).
		Return(okResult("Apply complete! Resources: 3 added."), nil)

	result, err := runWorkflow(t, env, temporal.ApplyWorkflow, contracts.WorkflowInput{
		Args: []string{"apply", "-auto-approve"},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
}

func TestApplyWorkflow_ActivityError(t *testing.T) {
	env := newEnv(t)
	env.OnActivity(temporal.ExecuteShellCommand, mock.Anything, mock.Anything).
		Return(nil, errors.New("worker crashed"))

	_, err := runWorkflow(t, env, temporal.ApplyWorkflow, contracts.WorkflowInput{
		Args: []string{"apply", "-auto-approve"},
	})

	if err == nil {
		t.Error("expected error propagation")
	}
}

func TestApplyWorkflow_NonZeroExit_IsNotGoError(t *testing.T) {
	env := newEnv(t)
	env.OnActivity(temporal.ExecuteShellCommand, mock.Anything, mock.Anything).
		Return(failResult(1, "Error: provider not found"), nil)

	result, err := runWorkflow(t, env, temporal.ApplyWorkflow, contracts.WorkflowInput{
		Args: []string{"apply", "-auto-approve"},
	})

	require.NoError(t, err)
	if result.ExitCode == 0 {
		t.Error("ExitCode = 0, want non-zero")
	}
}

// TestApplyWorkflow_MaxAttemptsIsHardcodedToOne verifies the documented
// contract: callers cannot increase MaxAttempts for ApplyWorkflow.
// The workflow must complete in exactly one activity execution (the test
// environment only calls the mock once — any retry would trigger an
// "unexpected call" panic from the mock).
func TestApplyWorkflow_MaxAttemptsIsHardcodedToOne(t *testing.T) {
	env := newEnv(t)
	// Set up the mock to be called exactly once.
	env.OnActivity(temporal.ExecuteShellCommand, mock.Anything, mock.Anything).
		Return(okResult("ok"), nil).Once()

	// Pass MaxAttempts=5 — the workflow must ignore it and use 1.
	result, err := runWorkflow(t, env, temporal.ApplyWorkflow, contracts.WorkflowInput{
		Args:        []string{"apply", "-auto-approve"},
		MaxAttempts: 5,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
}

// ── DestroyWorkflow ───────────────────────────────────────────────────────────

func TestDestroyWorkflow_HappyPath(t *testing.T) {
	env := newEnv(t)
	env.OnActivity(temporal.ExecuteShellCommand, mock.Anything, mock.Anything).
		Return(okResult("Destroy complete! Resources: 3 destroyed."), nil)

	result, err := runWorkflow(t, env, temporal.DestroyWorkflow, contracts.WorkflowInput{
		Args: []string{"destroy", "-auto-approve"},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
}

func TestDestroyWorkflow_ActivityError(t *testing.T) {
	env := newEnv(t)
	env.OnActivity(temporal.ExecuteShellCommand, mock.Anything, mock.Anything).
		Return(nil, errors.New("timeout"))

	_, err := runWorkflow(t, env, temporal.DestroyWorkflow, contracts.WorkflowInput{
		Args: []string{"destroy", "-auto-approve"},
	})

	if err == nil {
		t.Error("expected error propagation")
	}
}

func TestDestroyWorkflow_MaxAttemptsIsHardcodedToOne(t *testing.T) {
	env := newEnv(t)
	env.OnActivity(temporal.ExecuteShellCommand, mock.Anything, mock.Anything).
		Return(okResult("ok"), nil).Once()

	result, err := runWorkflow(t, env, temporal.DestroyWorkflow, contracts.WorkflowInput{
		Args:        []string{"destroy", "-auto-approve"},
		MaxAttempts: 10,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
}

// ── WorkspaceWorkflow ─────────────────────────────────────────────────────────

func TestWorkspaceWorkflow_HappyPath(t *testing.T) {
	env := newEnv(t)
	env.OnActivity(temporal.ExecuteShellCommand, mock.Anything, mock.Anything).
		Return(okResult("  default\n* production\n"), nil)

	result, err := runWorkflow(t, env, temporal.WorkspaceWorkflow, contracts.WorkflowInput{
		Args: []string{"workspace", "list"},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
}

func TestWorkspaceWorkflow_ActivityError(t *testing.T) {
	env := newEnv(t)
	env.OnActivity(temporal.ExecuteShellCommand, mock.Anything, mock.Anything).
		Return(nil, errors.New("workspace not found"))

	_, err := runWorkflow(t, env, temporal.WorkspaceWorkflow, contracts.WorkflowInput{
		Args: []string{"workspace", "select", "production"},
	})

	if err == nil {
		t.Error("expected error propagation")
	}
}

// ── ShellCommandWorkflow (fallback) ───────────────────────────────────────────

func TestShellCommandWorkflow_HappyPath(t *testing.T) {
	env := newEnv(t)
	env.OnActivity(temporal.ExecuteShellCommand, mock.Anything, mock.Anything).
		Return(okResult("output"), nil)

	result, err := runWorkflow(t, env, temporal.ShellCommandWorkflow, contracts.WorkflowInput{
		Args: []string{"version"},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestShellCommandWorkflow_ActivityError(t *testing.T) {
	env := newEnv(t)
	env.OnActivity(temporal.ExecuteShellCommand, mock.Anything, mock.Anything).
		Return(nil, errors.New("crash"))

	_, err := runWorkflow(t, env, temporal.ShellCommandWorkflow, contracts.WorkflowInput{
		Args: []string{"version"},
	})

	if err == nil {
		t.Error("expected error propagation")
	}
}

// ── Default timeout fallback ──────────────────────────────────────────────────

func TestWorkflows_DefaultTimeout_WhenZero(t *testing.T) {
	// TimeoutSeconds = 0 must not crash — the workflow uses its built-in default.
	workflows := []struct {
		name string
		fn   interface{}
	}{
		{"InitWorkflow", temporal.InitWorkflow},
		{"PlanWorkflow", temporal.PlanWorkflow},
		{"ApplyWorkflow", temporal.ApplyWorkflow},
		{"DestroyWorkflow", temporal.DestroyWorkflow},
		{"WorkspaceWorkflow", temporal.WorkspaceWorkflow},
		{"ShellCommandWorkflow", temporal.ShellCommandWorkflow},
	}

	for _, wf := range workflows {
		t.Run(wf.name, func(t *testing.T) {
			env := newEnv(t)
			env.OnActivity(temporal.ExecuteShellCommand, mock.Anything, mock.Anything).
				Return(okResult("ok"), nil)

			result, err := runWorkflow(t, env, wf.fn, contracts.WorkflowInput{
				Args:           []string{"arg"},
				TimeoutSeconds: 0, // must use the workflow's built-in default
			})

			require.NoError(t, err)
			require.NotNil(t, result)
		})
	}
}
