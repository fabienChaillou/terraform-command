// Package temporal contains the Temporal activity and workflow implementations
// for executing Terraform commands.
//
// This package is the only place in the system that knows which binary to run.
// The HTTP server never imports this package.
package temporal

import (
	"context"
	"os/exec"
	"strings"

	"go.temporal.io/sdk/activity"

	"github.com/fabienChaillou/terraform-contracts/contracts"
)

// TerraformBinary is the executable invoked by ExecuteShellCommand.
//
// It is intentionally a compile-time constant: the HTTP server must never be
// able to override the binary name.  Any change requires a worker deploy.
const TerraformBinary = "terraform"

// binaryName is the binary actually invoked at runtime.
// It equals TerraformBinary in production; tests may replace it with a
// stub (e.g. "echo") via the internal package variable — no exported setter
// is needed because tests live in the same package.
var binaryName = TerraformBinary

// ActivityInput is the payload passed from a workflow to the activity.
//
// The binary is absent — it is fixed by TerraformBinary above.
// Changing the binary requires a code change and redeployment, providing a
// natural audit gate for supply-chain security.
type ActivityInput struct {
	// Args are the CLI arguments forwarded verbatim to TerraformBinary.
	// Example: ["init", "-backend=true"] or ["apply", "-auto-approve"].
	Args []string `json:"args"`
}

// ExecuteShellCommand runs TerraformBinary with the provided arguments and
// returns the exit code, stdout, and stderr.
//
// A non-zero ExitCode is NOT a Go error — it signals a Terraform-level
// failure (e.g. plan diff exists, apply failed).  Go errors are reserved
// for infrastructure problems (binary missing, OS signals, timeouts).
//
// Activity heartbeat is driven by the ctx deadline set in the calling
// workflow's ActivityOptions; no explicit Heartbeat call is needed here
// unless the activity is long-running (handled per-workflow via HeartbeatTimeout).
func ExecuteShellCommand(ctx context.Context, input ActivityInput) (*contracts.ExecutionResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("ExecuteShellCommand: starting",
		"binary", binaryName,
		"args", strings.Join(input.Args, " "),
	)

	cmd := exec.CommandContext(ctx, binaryName, input.Args...)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	result := &contracts.ExecutionResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			// Non-zero exit from terraform — not an activity error.
			result.ExitCode = exitErr.ExitCode()
			logger.Info("ExecuteShellCommand: terraform exited non-zero",
				"exit_code", result.ExitCode,
				"stderr", result.Stderr,
			)
			return result, nil
		}
		// True infrastructure error (binary not found, killed by OS, etc.)
		logger.Error("ExecuteShellCommand: failed to run binary", "error", runErr)
		return nil, runErr
	}

	result.ExitCode = 0
	logger.Info("ExecuteShellCommand: success")
	return result, nil
}
