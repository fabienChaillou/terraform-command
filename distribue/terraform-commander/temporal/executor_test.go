// Unit tests for TemporalExecutor.
//
// No real Temporal server is required: client.Client is replaced by a
// mockWorkflowStarter that satisfies the narrow WorkflowStarter interface.
// Temporal's serialisation layer is bypassed — the mock returns Go values
// directly via a captured result pointer.
package temporal_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go.temporal.io/sdk/client"

	"github.com/fabienChaillou/terraform-commander/command"
	temporal "github.com/fabienChaillou/terraform-commander/temporal"
	"github.com/fabienChaillou/terraform-contracts/contracts"
)

// ── Test doubles ──────────────────────────────────────────────────────────────

// mockWorkflowRun implements client.WorkflowRun.
// Get immediately unmarshals resultValue into valuePtr (if non-nil) and
// returns runErr.
type mockWorkflowRun struct {
	resultValue *contracts.ExecutionResult
	runErr      error
}

func (m *mockWorkflowRun) GetID() string    { return "mock-workflow-id" }
func (m *mockWorkflowRun) GetRunID() string { return "mock-run-id" }
func (m *mockWorkflowRun) Get(_ context.Context, valuePtr interface{}) error {
	if m.runErr != nil {
		return m.runErr
	}
	if valuePtr != nil && m.resultValue != nil {
		// valuePtr is **contracts.ExecutionResult in the executor
		if pp, ok := valuePtr.(**contracts.ExecutionResult); ok {
			*pp = m.resultValue
		}
	}
	return nil
}

// mockWorkflowStarter records the last call to ExecuteWorkflow and returns
// the configured WorkflowRun / error.
type mockWorkflowStarter struct {
	run      client.WorkflowRun
	startErr error

	// captured fields for assertions
	capturedOptions  client.StartWorkflowOptions
	capturedWorkflow interface{}
	capturedInput    contracts.WorkflowInput
}

func (m *mockWorkflowStarter) ExecuteWorkflow(
	_ context.Context,
	opts client.StartWorkflowOptions,
	wf interface{},
	args ...interface{},
) (client.WorkflowRun, error) {
	m.capturedOptions = opts
	m.capturedWorkflow = wf
	if len(args) > 0 {
		if input, ok := args[0].(contracts.WorkflowInput); ok {
			m.capturedInput = input
		}
	}
	return m.run, m.startErr
}

// successStarter returns a starter whose workflow run yields exitCode 0.
func successStarter() *mockWorkflowStarter {
	return &mockWorkflowStarter{
		run: &mockWorkflowRun{
			resultValue: &contracts.ExecutionResult{ExitCode: 0, Stdout: "ok"},
		},
	}
}

// executorWith builds a TemporalExecutor wired to the given starter.
func executorWith(starter *mockWorkflowStarter) *temporal.TemporalExecutor {
	return &temporal.TemporalExecutor{
		Client:         starter,
		DefaultTimeout: 30 * time.Minute,
		MaxAttempts:    1,
		WorkflowByAction: contracts.ActionMap[string]{
			"init":      contracts.WorkflowInit,
			"plan":      contracts.WorkflowPlan,
			"apply":     contracts.WorkflowApply,
			"destroy":   contracts.WorkflowDestroy,
			"workspace": contracts.WorkflowWorkspace,
		},
	}
}

// ── resolveWorkflow ───────────────────────────────────────────────────────────

func TestResolveWorkflow_KnownActions(t *testing.T) {
	cases := []struct {
		action string
		want   string
	}{
		{"init", contracts.WorkflowInit},
		{"plan", contracts.WorkflowPlan},
		{"apply", contracts.WorkflowApply},
		{"destroy", contracts.WorkflowDestroy},
		{"workspace", contracts.WorkflowWorkspace},
	}

	for _, tc := range cases {
		starter := successStarter()
		exec := executorWith(starter)

		_, _ = exec.Execute(context.Background(), []string{tc.action}, command.ExecuteOptions{
			Action: tc.action,
		})

		if starter.capturedWorkflow != tc.want {
			t.Errorf("action %q: workflow = %v, want %q", tc.action, starter.capturedWorkflow, tc.want)
		}
	}
}

func TestResolveWorkflow_UnknownAction_FallsBackToShellCommand(t *testing.T) {
	starter := successStarter()
	exec := executorWith(starter)

	_, _ = exec.Execute(context.Background(), []string{"version"}, command.ExecuteOptions{
		Action: "version", // not in WorkflowByAction
	})

	if starter.capturedWorkflow != contracts.WorkflowShellCommand {
		t.Errorf("unknown action: workflow = %v, want %q",
			starter.capturedWorkflow, contracts.WorkflowShellCommand)
	}
}

func TestResolveWorkflow_EmptyAction_FallsBackToShellCommand(t *testing.T) {
	starter := successStarter()
	exec := executorWith(starter)

	_, _ = exec.Execute(context.Background(), []string{}, command.ExecuteOptions{Action: ""})

	if starter.capturedWorkflow != contracts.WorkflowShellCommand {
		t.Errorf("empty action: workflow = %v, want %q",
			starter.capturedWorkflow, contracts.WorkflowShellCommand)
	}
}

// ── Task queue ────────────────────────────────────────────────────────────────

func TestExecute_AllActionsUseSingleTaskQueue(t *testing.T) {
	actions := []string{"init", "plan", "apply", "destroy", "workspace"}

	for _, action := range actions {
		starter := successStarter()
		exec := executorWith(starter)

		_, _ = exec.Execute(context.Background(), []string{action}, command.ExecuteOptions{
			Action: action,
		})

		if starter.capturedOptions.TaskQueue != contracts.TaskQueue {
			t.Errorf("action %q: TaskQueue = %q, want %q",
				action, starter.capturedOptions.TaskQueue, contracts.TaskQueue)
		}
	}
}

// ── Execute — happy path ──────────────────────────────────────────────────────

func TestExecute_ReturnsResultFromWorkflow(t *testing.T) {
	starter := successStarter()
	exec := executorWith(starter)

	result, err := exec.Execute(context.Background(), []string{"plan"}, command.ExecuteOptions{
		Action: "plan",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
}

func TestExecute_ArgsForwardedToWorkflowInput(t *testing.T) {
	starter := successStarter()
	exec := executorWith(starter)

	wantArgs := []string{"plan", "-var", "env=prod", "-no-color"}
	_, _ = exec.Execute(context.Background(), wantArgs, command.ExecuteOptions{Action: "plan"})

	got := starter.capturedInput.Args
	if len(got) != len(wantArgs) {
		t.Fatalf("Args len = %d, want %d (got: %v)", len(got), len(wantArgs), got)
	}
	for i, a := range wantArgs {
		if got[i] != a {
			t.Errorf("Args[%d] = %q, want %q", i, got[i], a)
		}
	}
}

// ── Timeout resolution ────────────────────────────────────────────────────────

func TestExecute_ExplicitTimeoutForwarded(t *testing.T) {
	starter := successStarter()
	exec := executorWith(starter)

	_, _ = exec.Execute(context.Background(), []string{}, command.ExecuteOptions{
		Action:         "plan",
		TimeoutSeconds: 600,
	})

	if starter.capturedInput.TimeoutSeconds != 600 {
		t.Errorf("TimeoutSeconds = %d, want 600", starter.capturedInput.TimeoutSeconds)
	}
}

func TestExecute_ZeroTimeout_UsesDefaultTimeout(t *testing.T) {
	starter := successStarter()
	exec := executorWith(starter)
	exec.DefaultTimeout = 45 * time.Minute

	_, _ = exec.Execute(context.Background(), []string{}, command.ExecuteOptions{
		Action:         "plan",
		TimeoutSeconds: 0, // should fall back to DefaultTimeout
	})

	wantSec := int((45 * time.Minute).Seconds())
	if starter.capturedInput.TimeoutSeconds != wantSec {
		t.Errorf("TimeoutSeconds = %d, want %d (45 min)", starter.capturedInput.TimeoutSeconds, wantSec)
	}
}

// ── MaxAttempts resolution ────────────────────────────────────────────────────

func TestExecute_ExplicitMaxAttemptsForwarded(t *testing.T) {
	starter := successStarter()
	exec := executorWith(starter)

	_, _ = exec.Execute(context.Background(), []string{}, command.ExecuteOptions{
		Action:      "plan",
		MaxAttempts: 3,
	})

	if starter.capturedInput.MaxAttempts != 3 {
		t.Errorf("MaxAttempts = %d, want 3", starter.capturedInput.MaxAttempts)
	}
}

func TestExecute_ZeroMaxAttempts_UsesExecutorDefault(t *testing.T) {
	starter := successStarter()
	exec := executorWith(starter)
	exec.MaxAttempts = 2

	_, _ = exec.Execute(context.Background(), []string{}, command.ExecuteOptions{
		Action:      "plan",
		MaxAttempts: 0,
	})

	if starter.capturedInput.MaxAttempts != 2 {
		t.Errorf("MaxAttempts = %d, want 2 (executor default)", starter.capturedInput.MaxAttempts)
	}
}

func TestExecute_ZeroMaxAttempts_NoExecutorDefault_DefaultsToOne(t *testing.T) {
	starter := successStarter()
	exec := executorWith(starter)
	exec.MaxAttempts = 0 // no default set

	_, _ = exec.Execute(context.Background(), []string{}, command.ExecuteOptions{
		Action:      "apply",
		MaxAttempts: 0,
	})

	if starter.capturedInput.MaxAttempts != 1 {
		t.Errorf("MaxAttempts = %d, want 1 (hard minimum)", starter.capturedInput.MaxAttempts)
	}
}

// ── Execute — error paths ─────────────────────────────────────────────────────

func TestExecute_StartError_ReturnsWrappedError(t *testing.T) {
	starter := &mockWorkflowStarter{
		startErr: errors.New("temporal server unavailable"),
	}
	exec := executorWith(starter)

	result, err := exec.Execute(context.Background(), []string{}, command.ExecuteOptions{Action: "plan"})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result on start error, got %+v", result)
	}
}

func TestExecute_WorkflowError_ReturnsWrappedError(t *testing.T) {
	starter := &mockWorkflowStarter{
		run: &mockWorkflowRun{runErr: errors.New("workflow timed out")},
	}
	exec := executorWith(starter)

	result, err := exec.Execute(context.Background(), []string{}, command.ExecuteOptions{Action: "plan"})

	if err == nil {
		t.Fatal("expected error from workflow.Get, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result on workflow error, got %+v", result)
	}
}

// ── WorkflowID uniqueness ─────────────────────────────────────────────────────

func TestExecute_WorkflowID_ContainsActionName(t *testing.T) {
	starter := successStarter()
	exec := executorWith(starter)

	_, _ = exec.Execute(context.Background(), []string{}, command.ExecuteOptions{Action: "apply"})

	id := starter.capturedOptions.ID
	if id == "" {
		t.Fatal("workflow ID is empty")
	}
	// The ID must contain the action name for traceability.
	if !strings.Contains(id, "apply") {
		t.Errorf("workflow ID %q does not contain action name %q", id, "apply")
	}
}
