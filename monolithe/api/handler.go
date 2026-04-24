// Package api exposes the Terraform command endpoint via a Huma HTTP API.
//
// # Responsibilities
//
//  1. Resolve the action to a registered Command (via Resolver).
//  2. Validate the request arguments (via command.Validator).
//  3. Build the flat CLI argument slice (via command.ArgBuilder).
//  4. Forward the validated command to the Executor for remote execution.
//  5. Return the execution result or a structured validation error.
//
// The handler never executes a shell command itself, and has no knowledge of
// which binary is used — that is fixed by the activity's TerraformBinary constant.
//
// # Dependency Inversion
//
// Both dependencies are expressed as interfaces:
//   - Resolver   — abstracts the command registry (DIP).
//   - Executor   — abstracts the execution back-end (Temporal, local shell, …).
//
// main.go is the composition root that wires concrete types to these interfaces.
package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/fabienChaillou/terraform-commander/command"
)

// ── Dependency interfaces (DIP) ───────────────────────────────────────────────

// Resolver is the narrow interface the handler needs from the command registry.
// *command.Registry[Command] satisfies this interface via structural typing.
type Resolver interface {
	Lookup(action string) (command.Command, bool)
	GlobalHelp() string
}

// Executor is the single abstraction over how a validated command is dispatched
// for execution.
//
// The binary is intentionally absent from the signature: it is fixed by the
// TerraformBinary constant in the temporal/activity package and must not be
// configurable at the HTTP layer.
//
// Built-in implementation: *temporal.TemporalExecutor.
// Custom implementations can wrap any backend without modifying this package.
type Executor interface {
	Execute(
		ctx context.Context,
		args []string,
		opts command.ExecuteOptions,
	) (*command.ExecutionResult, error)
}

// ── Huma I/O types ────────────────────────────────────────────────────────────

// CommandInput is the JSON body for POST /terraform.
type CommandInput struct {
	Body command.Request
}

// CommandOutput is the JSON response for a successfully dispatched command.
// A non-zero ExitCode means Terraform itself reported an error; the HTTP
// status is still 200 because the API call (validate + dispatch) succeeded.
type CommandOutput struct {
	Body *command.ExecutionResult
}

// ── Route configuration ───────────────────────────────────────────────────────

// Config holds per-action execution tuning knobs.
//
// TimeoutByAction and MaxAttemptsByAction use ActionMap[T] so that absent
// keys return an explicit default instead of silently producing zero values.
type Config struct {
	// TimeoutByAction maps action names to execution timeout in seconds.
	TimeoutByAction command.ActionMap[int]

	// MaxAttemptsByAction maps action names to their maximum retry count.
	// Apply and destroy default to 1 (no retry) — they are not idempotent.
	MaxAttemptsByAction command.ActionMap[int32]
}

func defaultConfig() Config {
	return Config{
		TimeoutByAction: command.ActionMap[int]{
			"apply":     1800, // 30 min
			"destroy":   1800,
			"plan":      900, // 15 min
			"init":      300, // 5 min
			"workspace": 60,  // 1 min
		},
		MaxAttemptsByAction: command.ActionMap[int32]{
			// apply and destroy are NOT idempotent — never retry automatically.
			"apply":   1,
			"destroy": 1,
			// idempotent actions — safe to retry on transient OS failures.
			"plan":      3,
			"init":      3,
			"workspace": 2,
		},
	}
}

// ── Route registration ────────────────────────────────────────────────────────

// RegisterRoutes wires the POST /terraform operation into the Huma API.
//
//	resolver  — resolves action names to Command implementations.
//	executor  — dispatches the validated command for execution.
//	cfg       — optional Config overrides; pass nil to use defaults.
func RegisterRoutes(
	humaAPI huma.API,
	resolver Resolver,
	executor Executor,
	cfg *Config,
) {
	if cfg == nil {
		c := defaultConfig()
		cfg = &c
	}

	huma.Register(humaAPI, huma.Operation{
		OperationID: "run-terraform-command",
		Method:      http.MethodPost,
		Path:        "/terraform",
		Summary:     "Run a Terraform command",
		Description: "Validates the request on the server, then forwards the " +
			"command to a worker for shell execution via Temporal.",
		Tags: []string{"terraform"},
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnprocessableEntity,
			http.StatusInternalServerError,
		},
	}, func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		req := &input.Body

		// ── Step 1: resolve the command ───────────────────────────────────────
		cmd, ok := resolver.Lookup(req.Action)
		if !ok {
			detail := huma.ErrorDetail{
				Message:  formatUnknownAction(req.Action, resolver),
				Location: "body.action",
			}
			return nil, huma.Error400BadRequest(resolver.GlobalHelp(), &detail)
		}

		// ── Step 2: validate arguments ────────────────────────────────────────
		if errs := cmd.Validate(req); len(errs) > 0 {
			details := make([]error, len(errs))
			for i, e := range errs {
				details[i] = &huma.ErrorDetail{
					Message:  e.Message,
					Location: "body.args." + e.Field,
				}
			}
			if isSubcmdError(errs) {
				return nil, huma.Error400BadRequest(cmd.Help(), details...)
			}
			return nil, huma.Error422UnprocessableEntity(cmd.Help(), details...)
		}

		// ── Step 3: build CLI args ────────────────────────────────────────────
		args := cmd.BuildArgs(req)

		// ── Step 4: dispatch ──────────────────────────────────────────────────
		opts := command.ExecuteOptions{
			Action:         req.Action,
			TimeoutSeconds: cfg.TimeoutByAction.Get(req.Action, 0),
			MaxAttempts:    cfg.MaxAttemptsByAction.Get(req.Action, 1),
		}

		result, err := executor.Execute(ctx, args, opts)
		if err != nil {
			return nil, huma.Error500InternalServerError("execution error", err)
		}

		return &CommandOutput{Body: result}, nil
	})
}

// ── helpers ───────────────────────────────────────────────────────────────────

func formatUnknownAction(action string, r Resolver) string {
	if action == "" {
		return "action is required"
	}
	return "unknown action: " + action
}

func isSubcmdError(errs []command.ValidationError) bool {
	for _, e := range errs {
		if e.Field != "subcommand" {
			return false
		}
	}
	return true
}
