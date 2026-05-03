// Package worker wires Temporal workflows and activities for every Terraform
// command. Each action maps to a dedicated workflow type (InitWorkflow,
// PlanWorkflow, WorkspaceWorkflow …) that executes a single activity
// (ExecTerraformActivity) via os/exec.
package worker

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/fabienChaillou/terraform-cmd/internal/terraform"
)

// ─── Constants ────────────────────────────────────────────────────────────────

const (
	// TaskQueue is the single Temporal task queue used by all terraform workers.
	TaskQueue = "terraform-task-queue"

	// ActivityName is the unique name of the exec activity shared by all workflows.
	ActivityName = "ExecTerraformActivity"
)

// WorkflowName returns the Temporal workflow type name for a given command.
// e.g. "init" → "InitWorkflow", "workspace" → "WorkspaceWorkflow"
func WorkflowName(action string) string {
	if action == "" {
		return "UnknownWorkflow"
	}
	return strings.ToUpper(action[:1]) + action[1:] + "Workflow"
}

// ─── Activity input / output ──────────────────────────────────────────────────

// ExecInput is the serialisable input passed from the workflow to the activity.
type ExecInput struct {
	// Args is the full terraform CLI argument list, e.g. ["workspace", "new", "staging"]
	Args []string `json:"args"`
	// WorkDir is an optional working directory for the terraform process.
	WorkDir string `json:"work_dir,omitempty"`
}

// ExecOutput holds the captured output of the terraform process.
type ExecOutput struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}

// ─── Payload → CLI args builder ───────────────────────────────────────────────

// BuildArgs converts a validated CommandResult into a terraform CLI argument
// slice ready to be passed to exec.Command("terraform", args...).
func BuildArgs(result terraform.CommandResult) []string {
	args := []string{result.Command}

	// Sub-command (workspace list, state mv …)
	if result.SubCommand != "" {
		args = append(args, result.SubCommand)
	}

	p := result.Payload

	// Boolean flags → -flag or -flag=value
	boolFlags := map[string]string{
		"auto_approve":  "-auto-approve",
		"backend":       "-backend=true",
		"upgrade":       "-upgrade",
		"reconfigure":   "-reconfigure",
		"migrate_state": "-migrate-state",
		"json":          "-json",
		"no_color":      "-no-color",
		"destroy":       "-destroy",
		"refresh":       "-refresh=true",
		"check":         "-check",
		"diff":          "-diff",
		"recursive":     "-recursive",
		"raw":           "-raw",
		"draw_cycles":   "-draw-cycles",
		"force":         "-force",
		"lock":          "-lock=true",
	}
	for key, flag := range boolFlags {
		if v, ok := p[key].(bool); ok && v {
			args = append(args, flag)
		}
	}

	// String flags → -flag=value
	stringFlags := map[string]string{
		"out":            "-out",
		"var_file":       "-var-file",
		"backend_config": "-backend-config",
		"plan":           "-plan",
		"path":           "", // positional for "show"
		"state":          "-state",
		"type":           "-type",
	}
	for key, flag := range stringFlags {
		if v, ok := p[key].(string); ok && v != "" {
			if flag == "" {
				args = append(args, v) // positional
			} else {
				args = append(args, fmt.Sprintf("%s=%s", flag, v))
			}
		}
	}

	// Parallelism
	if v, ok := p["parallelism"].(int); ok && v > 0 {
		args = append(args, fmt.Sprintf("-parallelism=%d", v))
	}

	// Repeated -target flags
	if targets, ok := p["target"].([]string); ok {
		for _, t := range targets {
			args = append(args, fmt.Sprintf("-target=%s", t))
		}
	}

	// Repeated -var flags
	if vars, ok := p["var"].([]string); ok {
		for _, v := range vars {
			args = append(args, fmt.Sprintf("-var=%s", v))
		}
	}

	// Positional: workspace name, state address / destination, import address+id, dir
	for _, key := range []string{"name", "address", "destination", "id", "dir"} {
		if v, ok := p[key].(string); ok && v != "" {
			args = append(args, v)
		}
	}

	return args
}

// ─── Activity ─────────────────────────────────────────────────────────────────

// Activities holds the activity implementations (receiver keeps them groupable).
type Activities struct {
	// TerraformBin is the path to the terraform binary (default "terraform").
	TerraformBin string
}

func NewActivities(bin string) *Activities {
	if bin == "" {
		bin = "terraform"
	}
	return &Activities{TerraformBin: bin}
}

// ExecTerraformActivity runs terraform with the provided CLI args.
// It is the single activity shared by all workflows.
func (a *Activities) ExecTerraformActivity(ctx context.Context, input ExecInput) (ExecOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("ExecTerraformActivity", "args", input.Args, "workDir", input.WorkDir)

	//nolint:gosec // args are validated upstream by the dispatcher
	cmd := exec.CommandContext(ctx, a.TerraformBin, input.Args...)

	if input.WorkDir != "" {
		cmd.Dir = input.WorkDir
	}

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	out := ExecOutput{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: 0,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			out.ExitCode = exitErr.ExitCode()
			// Non-zero exit is surfaced as a Temporal ApplicationError so the
			// workflow can decide whether to retry.
			return out, temporal.NewApplicationError(
				fmt.Sprintf("terraform exited with code %d: %s", out.ExitCode, out.Stderr),
				"TerraformExitError",
				out,
			)
		}
		return out, fmt.Errorf("exec failed: %w", err)
	}

	logger.Info("ExecTerraformActivity done", "exitCode", out.ExitCode)
	return out, nil
}

// ─── Generic workflow factory ─────────────────────────────────────────────────

// WorkflowInput is the serialisable input to every terraform workflow.
type WorkflowInput struct {
	Result  terraform.CommandResult `json:"result"`
	WorkDir string                  `json:"work_dir,omitempty"`
}

// WorkflowOutput is returned by every terraform workflow.
type WorkflowOutput struct {
	WorkflowName string     `json:"workflow_name"`
	Action       string     `json:"action"`
	SubCommand   string     `json:"sub_command,omitempty"`
	Exec         ExecOutput `json:"exec"`
}

// makeTerraformWorkflow returns a workflow function bound to a specific action
// name. The returned func satisfies the Temporal workflow function signature.
//
// Each generated workflow:
//  1. Logs the action being executed.
//  2. Builds the CLI args from the validated CommandResult.
//  3. Schedules ExecTerraformActivity with standard retry options.
//  4. Returns the captured output.
func makeTerraformWorkflow(action string) func(workflow.Context, WorkflowInput) (WorkflowOutput, error) {
	return func(ctx workflow.Context, input WorkflowInput) (WorkflowOutput, error) {
		logger := workflow.GetLogger(ctx)
		wfName := WorkflowName(action)
		logger.Info(wfName+" started", "action", action, "subCommand", input.Result.SubCommand)

		ao := workflow.ActivityOptions{
			StartToCloseTimeout: 10 * time.Minute,
			RetryPolicy: &temporal.RetryPolicy{
				InitialInterval:    5 * time.Second,
				BackoffCoefficient: 2.0,
				MaximumInterval:    1 * time.Minute,
				MaximumAttempts:    3,
				// Do not retry on explicit terraform exit errors (bad config etc.)
				NonRetryableErrorTypes: []string{"TerraformExitError"},
			},
		}
		ctx = workflow.WithActivityOptions(ctx, ao)

		execInput := ExecInput{
			Args:    BuildArgs(input.Result),
			WorkDir: input.WorkDir,
		}

		var execOut ExecOutput
		if err := workflow.ExecuteActivity(ctx, ActivityName, execInput).Get(ctx, &execOut); err != nil {
			return WorkflowOutput{}, fmt.Errorf("%s activity failed: %w", wfName, err)
		}

		return WorkflowOutput{
			WorkflowName: wfName,
			Action:       action,
			SubCommand:   input.Result.SubCommand,
			Exec:         execOut,
		}, nil
	}
}

// ─── Worker factory ───────────────────────────────────────────────────────────

// WorkerFactory builds and manages Temporal workers, one per registered action.
type WorkerFactory struct {
	client     client.Client
	registry   *terraform.Registry
	activities *Activities
	workers    []worker.Worker
}

func NewWorkerFactory(c client.Client, r *terraform.Registry, terraformBin string) *WorkerFactory {
	return &WorkerFactory{
		client:     c,
		registry:   r,
		activities: NewActivities(terraformBin),
	}
}

// Start registers every terraform command as a Temporal workflow type on a
// single shared task queue, then starts the worker.
//
// All workflows share one worker + one task queue. The workflow type names
// (InitWorkflow, PlanWorkflow …) are derived dynamically from the command names.
func (f *WorkerFactory) Start() error {
	w := worker.New(f.client, TaskQueue, worker.Options{})

	// Register the shared activity once
	w.RegisterActivityWithOptions(
		f.activities.ExecTerraformActivity,
		activity.RegisterOptions{Name: ActivityName},
	)

	// Register one workflow per action
	for _, action := range f.registry.Actions() {
		wfName := WorkflowName(action)
		wfFunc := makeTerraformWorkflow(action)
		w.RegisterWorkflowWithOptions(wfFunc, workflow.RegisterOptions{Name: wfName})
	}

	f.workers = append(f.workers, w)
	return w.Start()
}

// Stop gracefully stops all workers.
func (f *WorkerFactory) Stop() {
	for _, w := range f.workers {
		w.Stop()
	}
}

// ─── Executor ─────────────────────────────────────────────────────────────────

// Executor submits a validated CommandResult as a Temporal workflow execution.
type Executor struct {
	client client.Client
}

func NewExecutor(c client.Client) *Executor { return &Executor{client: c} }

// Execute starts a Temporal workflow for the given CommandResult and waits for
// the result (synchronous for simplicity — callers can switch to async via
// ExecuteAsync if needed).
func (e *Executor) Execute(ctx context.Context, result terraform.CommandResult, workDir string) (WorkflowOutput, error) {
	wfName := WorkflowName(result.Command)

	options := client.StartWorkflowOptions{
		// Workflow ID is deterministic to allow deduplication within a time window
		ID:        fmt.Sprintf("terraform-%s-%d", result.Command, time.Now().UnixNano()),
		TaskQueue: TaskQueue,
	}

	input := WorkflowInput{Result: result, WorkDir: workDir}

	run, err := e.client.ExecuteWorkflow(ctx, options, wfName, input)
	if err != nil {
		return WorkflowOutput{}, fmt.Errorf("start workflow %s: %w", wfName, err)
	}

	var out WorkflowOutput
	if err := run.Get(ctx, &out); err != nil {
		return WorkflowOutput{}, fmt.Errorf("workflow %s failed: %w", wfName, err)
	}

	return out, nil
}

// ExecuteAsync starts the workflow without waiting for completion.
// Returns the workflow run so the caller can poll later.
func (e *Executor) ExecuteAsync(ctx context.Context, result terraform.CommandResult, workDir string) (client.WorkflowRun, error) {
	wfName := WorkflowName(result.Command)

	options := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("terraform-%s-%d", result.Command, time.Now().UnixNano()),
		TaskQueue: TaskQueue,
	}

	return e.client.ExecuteWorkflow(ctx, options, wfName, WorkflowInput{
		Result:  result,
		WorkDir: workDir,
	})
}
