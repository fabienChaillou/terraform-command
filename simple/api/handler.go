// Package api exposes a single POST /terraform/command route built with
// the Huma v2 framework (https://huma.rocks).
//
// Huma gives us:
//   - Automatic JSON schema generation & request validation from Go structs.
//   - OpenAPI 3.1 spec at /openapi.json (and Swagger UI at /docs).
//   - Typed, structured error responses that follow RFC 9457 (Problem Details).
//   - Zero boilerplate for Content-Type negotiation and status codes.
package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/fabienChaillou/terraform-cmd/internal/terraform"
	"github.com/fabienChaillou/terraform-cmd/internal/worker"
)

// ─── Request / Response types ─────────────────────────────────────────────────

// CommandRequest is the typed body of POST /terraform/command.
// Every field is optional at the struct level; the Dispatcher applies the
// per-action validation rules after the whitelist check.
type CommandRequest struct {
	// Action is the Terraform command name (required).
	Action string `json:"action" doc:"Terraform command (init, plan, apply, destroy, workspace, state, …)" required:"true" minLength:"1"`

	// SubCommand is used by commands that have sub-commands (workspace, state).
	SubCommand string `json:"sub_command,omitempty" doc:"Sub-command (e.g. list, new, select for workspace)"`

	// WorkDir is forwarded to the Temporal activity as the working directory.
	WorkDir string `json:"work_dir,omitempty" doc:"Optional working directory for the terraform process"`

	// --- common terraform flags ---
	AutoApprove   bool     `json:"auto_approve,omitempty"    doc:"Skip interactive approval (-auto-approve)"`
	Backend       bool     `json:"backend,omitempty"         doc:"Configure backend (-backend=true)"`
	BackendConfig string   `json:"backend_config,omitempty"  doc:"Backend config file (-backend-config=path)"`
	Check         bool     `json:"check,omitempty"           doc:"Check formatting only (-check)"`
	Destroy       bool     `json:"destroy,omitempty"         doc:"Plan a destroy (-destroy)"`
	Diff          bool     `json:"diff,omitempty"            doc:"Show formatting diffs (-diff)"`
	Dir           string   `json:"dir,omitempty"             doc:"Working directory"`
	DrawCycles    bool     `json:"draw_cycles,omitempty"     doc:"Highlight dependency cycles (-draw-cycles)"`
	Force         bool     `json:"force,omitempty"           doc:"Force deletion (-force)"`
	GraphType     string   `json:"type,omitempty"            doc:"Graph type (plan|plan-refresh-only|plan-destroy|apply)"`
	ID            string   `json:"id,omitempty"              doc:"Provider-specific resource ID (import)"`
	JSON          bool     `json:"json,omitempty"            doc:"Machine-readable JSON output (-json)"`
	Lock          bool     `json:"lock,omitempty"            doc:"Lock state (-lock=true)"`
	MigrateState  bool     `json:"migrate_state,omitempty"   doc:"Migrate state (-migrate-state)"`
	Name          string   `json:"name,omitempty"            doc:"Workspace name"`
	NoColor       bool     `json:"no_color,omitempty"        doc:"Disable terminal colors (-no-color)"`
	Out           string   `json:"out,omitempty"             doc:"Write plan to file (-out=path)"`
	Parallelism   int      `json:"parallelism,omitempty"     doc:"Parallel resource operations (1-100)" minimum:"1" maximum:"100"`
	Path          string   `json:"path,omitempty"            doc:"Path to plan file (show)"`
	Plan          string   `json:"plan,omitempty"            doc:"Pre-computed plan file (apply)"`
	Raw           bool     `json:"raw,omitempty"             doc:"Print raw string value (-raw)"`
	Reconfigure   bool     `json:"reconfigure,omitempty"     doc:"Reconfigure backend (-reconfigure)"`
	Recursive     bool     `json:"recursive,omitempty"       doc:"Process sub-directories (-recursive)"`
	Refresh       bool     `json:"refresh,omitempty"         doc:"Refresh state before planning (-refresh=true)"`
	Address       string   `json:"address,omitempty"         doc:"Resource address (state / import)"`
	Destination   string   `json:"destination,omitempty"     doc:"Destination address (state mv)"`
	State         string   `json:"state,omitempty"           doc:"State file path (workspace new -state=path)"`
	Target        []string `json:"target,omitempty"          doc:"Limit scope to specific resources (-target=…)"`
	Upgrade       bool     `json:"upgrade,omitempty"         doc:"Upgrade providers and modules (-upgrade)"`
	Var           []string `json:"var,omitempty"             doc:"Variable values (-var=k=v)"`
	VarFile       string   `json:"var_file,omitempty"        doc:"Variable file (-var-file=path)"`
	OutputName    string   `json:"output_name,omitempty"     doc:"Specific output name (output command)"`
}

// toPayload converts the typed request to the generic map expected by the
// Dispatcher. Only non-zero fields are included so Validate() logic stays
// intact (absent key ≠ false / empty string).
func (r *CommandRequest) toPayload() map[string]any {
	p := map[string]any{"action": r.Action}

	setStr := func(k, v string) {
		if v != "" {
			p[k] = v
		}
	}
	setBool := func(k string, v bool) {
		if v {
			p[k] = v
		}
	}
	setStrs := func(k string, v []string) {
		if len(v) > 0 {
			p[k] = v
		}
	}

	setStr("sub_command", r.SubCommand)
	setStr("backend_config", r.BackendConfig)
	setStr("dir", r.Dir)
	setStr("id", r.ID)
	setStr("name", r.Name)
	setStr("out", r.Out)
	setStr("path", r.Path)
	setStr("plan", r.Plan)
	setStr("address", r.Address)
	setStr("destination", r.Destination)
	setStr("state", r.State)
	setStr("type", r.GraphType)
	setStr("var_file", r.VarFile)
	setStr("output_name", r.OutputName)

	setBool("auto_approve", r.AutoApprove)
	setBool("backend", r.Backend)
	setBool("check", r.Check)
	setBool("destroy", r.Destroy)
	setBool("diff", r.Diff)
	setBool("draw_cycles", r.DrawCycles)
	setBool("force", r.Force)
	setBool("json", r.JSON)
	setBool("lock", r.Lock)
	setBool("migrate_state", r.MigrateState)
	setBool("no_color", r.NoColor)
	setBool("raw", r.Raw)
	setBool("reconfigure", r.Reconfigure)
	setBool("recursive", r.Recursive)
	setBool("refresh", r.Refresh)
	setBool("upgrade", r.Upgrade)

	setStrs("target", r.Target)
	setStrs("var", r.Var)

	if r.Parallelism > 0 {
		p["parallelism"] = r.Parallelism
	}

	return p
}

// CommandInput is the Huma input wrapper (body lives in the Body field).
type CommandInput struct {
	Body *CommandRequest
}

// CommandOutput is the Huma output wrapper (status 200).
type CommandOutput struct {
	Body *CommandResponseBody
}

// CommandResponseBody is the JSON body returned on success.
type CommandResponseBody struct {
	Result   terraform.CommandResult `json:"result"`
	Workflow *worker.WorkflowOutput  `json:"workflow,omitempty"`
}

// ─── Router builder ───────────────────────────────────────────────────────────

// NewRouter creates a chi router with the Huma API mounted on it.
//
// Exposed routes:
//   - POST /terraform/command  — main endpoint
//   - GET  /openapi.json       — OpenAPI 3.1 spec (auto-generated by Huma)
//   - GET  /docs               — Swagger UI (auto-generated by Huma)
func NewRouter(d *terraform.Dispatcher, e *worker.Executor) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)

	humaAPI := humachi.New(r, huma.DefaultConfig(
		"Terraform Command API",
		"1.0.0",
	))

	registerCommandRoute(humaAPI, d, e)
	return r
}

// NewRouterDryRun creates a router without an Executor (validate-only mode).
// Useful for tests and development without a live Temporal cluster.
func NewRouterDryRun(d *terraform.Dispatcher) http.Handler {
	return NewRouter(d, nil)
}

// ─── Route registration ───────────────────────────────────────────────────────

func registerCommandRoute(humaAPI huma.API, d *terraform.Dispatcher, e *worker.Executor) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "run-terraform-command",
		Method:      http.MethodPost,
		Path:        "/terraform/command",
		Summary:     "Execute a Terraform command",
		Description: `Validates the payload against the action's rules and, when an
Executor is configured, starts a Temporal workflow that runs terraform via exec.Command.

**Pipeline**
1. Huma validates required fields and JSON types (schema-level).
2. Dispatcher runs whitelist check + per-action validation.
3. On error → 422 Unprocessable Entity with problem details + help text.
4. On success → optional Temporal workflow execution → 200 OK.`,
		Tags: []string{"terraform"},
		// DefaultStatus is 200; Huma handles Content-Type automatically.
	}, func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		payload := input.Body.toPayload()

		// ── 1. Dispatcher: whitelist + per-action validation ─────────────────
		result := d.Dispatch(payload)
		if !result.Valid {
			detail := buildErrorDetail(result.HelpText, result.Errors)
			return nil, huma.Error422UnprocessableEntity(detail)
		}

		resp := &CommandOutput{Body: &CommandResponseBody{Result: result}}

		// ── 2. Temporal execution (optional) ─────────────────────────────────
		if e != nil {
			wfOut, err := e.Execute(ctx, result, input.Body.WorkDir)
			if err != nil {
				return nil, huma.Error500InternalServerError(
					"workflow execution failed: " + err.Error(),
				)
			}
			resp.Body.Workflow = &wfOut
		}

		return resp, nil
	})
}

// buildErrorDetail assembles a human-readable error detail string that combines
// the per-field validation errors with the command's help text.
func buildErrorDetail(help string, errs []terraform.ValidationError) string {
	msg := "validation failed"
	for _, e := range errs {
		msg += "; " + e.Error()
	}
	if help != "" {
		msg += "\n\n" + help
	}
	return msg
}
