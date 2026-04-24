// Package temporal provides the Temporal workflow and activity for Terraform
// command execution.
//
// The binary to execute is fixed as a package-level constant (TerraformBinary).
// All workflows and the activity share this constant — there is no need to
// propagate the binary name through WorkflowInput or ActivityInput.
package temporal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"

	"go.temporal.io/sdk/activity"

	"github.com/fabienChaillou/terraform-commander/command"
)

// TerraformBinary is the executable name used by every activity in this package.
// Change this constant (or override it at build time with -ldflags) to switch
// to an alternative distribution such as OpenTofu ("tofu").
const TerraformBinary = "terraform"

// ── Activity input ────────────────────────────────────────────────────────────

// ActivityInput is the JSON-serialisable payload sent from a workflow to the
// ExecuteShellCommand activity.
//
// The binary is intentionally absent: it is fixed by TerraformBinary so that
// the worker cannot be instructed to run an arbitrary executable.
type ActivityInput struct {
	// Args is the flat argument slice built by command.Command.BuildArgs on the
	// server side, e.g. ["plan", "-var", "env=prod", "-out", "plan.out"].
	Args []string `json:"args"`
}

// ── Activity implementation ───────────────────────────────────────────────────

// ExecuteShellCommand is the Temporal activity that runs TerraformBinary with
// the given Args and captures its output.
//
// Design decisions:
//   - A non-zero exit code is a VALID result, not a Go error.
//     The caller (api layer) decides what to do with exit_code != 0.
//   - A Go error is only returned for OS-level failures (binary not found,
//     permission denied, context cancelled/timed out).
//   - The activity does NOT retry on non-zero exit codes because Terraform
//     errors (e.g. missing provider, state conflict) are not transient.
//     Retry behaviour is controlled by each workflow's RetryPolicy.
func ExecuteShellCommand(
	ctx context.Context,
	input ActivityInput,
) (*command.ExecutionResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("ExecuteShellCommand: starting",
		"binary", TerraformBinary,
		"args", input.Args,
	)

	cmd := exec.CommandContext(ctx, TerraformBinary, input.Args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	// Happy path: exit 0.
	if runErr == nil {
		logger.Info("ExecuteShellCommand: completed", "exit_code", 0)
		return &command.ExecutionResult{
			ExitCode: 0,
			Stdout:   stdout.String(),
			Stderr:   stderr.String(),
		}, nil
	}

	// Non-zero exit: Terraform reported an error — return the result, not a Go error.
	var exitErr *exec.ExitError
	if errors.As(runErr, &exitErr) {
		code := exitErr.ExitCode()
		logger.Info("ExecuteShellCommand: non-zero exit", "exit_code", code)
		return &command.ExecutionResult{
			ExitCode: code,
			Stdout:   stdout.String(),
			Stderr:   stderr.String(),
		}, nil
	}

	// OS-level failure (binary not found, context cancelled, etc.).
	// Return a Go error so Temporal can apply the workflow's RetryPolicy.
	logger.Error("ExecuteShellCommand: OS error", "error", runErr)
	return nil, fmt.Errorf("exec %s: %w", TerraformBinary, runErr)
}
