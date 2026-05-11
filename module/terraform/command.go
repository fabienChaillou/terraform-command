// Package terraform defines the core contracts shared by every component of
// the Terraform Commander platform: the Command interface family, the request
// and validation-error types, the generic Registry, and the execution result
// types.
//
// Concrete implementations of the Command interface (init, plan, apply,
// destroy, workspace) live in the sub-package
// github.com/fabienChaillou/terraform-commander/internal/terraform/command.
//
// # Design
//
// The interface hierarchy follows the Interface Segregation Principle (ISP):
// three focused interfaces — Validator, ArgBuilder, HelpProvider — are composed
// into the omnibus Command interface.  Callers that only need validation can
// depend on Validator alone; callers that need everything depend on Command.
//
//	Validator ──┐
//	ArgBuilder ─┼──► Command
//	HelpProvider┘
//
// No type in this package ever touches a shell, a file, or a network socket.
// Execution is the worker's responsibility.
package terraform

// ── Shared request / error types ─────────────────────────────────────────────

// Request is the JSON body sent to POST /terraform.
type Request struct {
	// Action is the top-level terraform command: init | plan | apply | destroy | workspace.
	Action string `json:"action"`

	// Subcommand is only used by commands that have sub-commands (e.g. workspace).
	Subcommand string `json:"subcommand,omitempty"`

	// Args are the key/value pairs that map to CLI flags.
	// Values can be strings, booleans, or maps (for -var).
	//   {"auto-approve": true, "var": {"env": "prod"}, "target": "aws_vpc.main"}
	Args map[string]interface{} `json:"args,omitempty"`
}

// ValidationError describes a single invalid field in a Request.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ── Focused interfaces (ISP) ──────────────────────────────────────────────────

// Validator checks a request for structural and semantic errors.
// An empty slice means the request is valid.
//
// Callers that only perform validation (e.g. a dry-run middleware) should
// depend on this interface rather than the full Command.
type Validator interface {
	Validate(req *Request) []ValidationError
}

// ArgBuilder converts a validated Request into a flat CLI argument slice
// ready to be passed verbatim to exec.Command.
//
// Example output for `terraform plan -var env=prod -target aws_vpc.main`:
//
//	[]string{"plan", "-var", "env=prod", "-target", "aws_vpc.main"}
//
// BuildArgs must only be called after Validate returns no errors.
type ArgBuilder interface {
	BuildArgs(req *Request) []string
}

// HelpProvider returns human-readable usage text for a command.
// The text is included in the API error response when validation fails so
// the caller knows how to correct the request.
type HelpProvider interface {
	Help() string
}

// ── Composite interface ───────────────────────────────────────────────────────

// Command is the full contract every terraform sub-command must implement.
// It composes Validator, ArgBuilder, and HelpProvider.
//
// Concrete commands live in the sub-package
// github.com/fabienChaillou/terraform-commander/internal/terraform/command
// (InitCommand, PlanCommand, etc.).  Custom commands registered via
// Registry.Register must also satisfy Command (or the more specific type
// parameter of a Registry[C]).
type Command interface {
	Validator
	ArgBuilder
	HelpProvider
}
