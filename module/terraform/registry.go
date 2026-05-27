package terraform

import (
	"strings"
)

// GlobalHelp is the top-level usage text returned by Registry.GlobalHelp.
//
// It is exported as a package-level constant so callers can override or
// extend it (e.g. when adding custom actions) without touching the registry
// implementation.
const GlobalHelp = `Usage: terraform [global options] <subcommand> [args]

Main commands:
  init        Prepare your working directory for other commands
  plan        Show changes required by the current configuration
  apply       Create or update infrastructure
  destroy     Destroy previously-created infrastructure
  refresh     Update local state to match real infrastructure
  output      Read output values from a Terraform state file

State management:
  state       Advanced state management (subcommands: list | show | mv | rm | pull | push)
  taint       Mark a resource for recreation on next apply
  untaint     Remove the "tainted" marker from a resource
  unlock      Force-release a state lock (CLI verb: force-unlock)

Workspace management:
  workspace   Workspace management (subcommands: new | select | list | delete | show)

Use "terraform <command> --help" for more information about a command.
`

// ── Generic Registry ──────────────────────────────────────────────────────────

// Registry[C] maps action names to Command implementations of type C.
//
// The type parameter C must satisfy the Command interface, but can be a more
// specific sub-interface or concrete type — enabling type-safe registries for
// domain-specific command sets (e.g. Registry[TerraformCommand]) without
// modifying this package.
//
// # Open/Closed Principle
//
// The registry is open for extension (Register any C at runtime) but closed
// for modification: new command sets are added by calling Register or by
// constructing a new Registry[C] via NewRegistryOf[C], never by editing this
// file.
//
// # Usage
//
//	// Generic, empty registry — populate it yourself:
//	r := terraform.NewRegistryOf[terraform.Command]()
//
//	// Pre-loaded with the five built-in actions — see the command sub-package:
//	r := command.NewRegistry()
//
//	// Typed registry for a specialised interface:
//	type TFCommand interface { terraform.Command; Timeout() int }
//	r := terraform.NewRegistryOf[TFCommand]()
type Registry[C Command] struct {
	commands map[string]C
}

// NewRegistryOf returns an empty Registry whose command type is C.
// Use this when you need a registry constrained to a more specific interface,
// or when you want to populate the registry yourself.
//
//	type TFCommand interface { terraform.Command; Timeout() int }
//	r := terraform.NewRegistryOf[TFCommand]()
func NewRegistryOf[C Command]() *Registry[C] {
	return &Registry[C]{commands: make(map[string]C)}
}

// Register adds or replaces a command.  name is normalised to lower-case.
func (r *Registry[C]) Register(name string, cmd C) {
	r.commands[strings.ToLower(name)] = cmd
}

// Lookup returns the command registered for action (case-insensitive, trimmed).
// The second return value is false if no command is registered for that action.
func (r *Registry[C]) Lookup(action string) (C, bool) {
	cmd, ok := r.commands[strings.ToLower(strings.TrimSpace(action))]
	return cmd, ok
}

// GlobalHelp returns the top-level usage text shown when the action is
// missing or unknown.
func (r *Registry[C]) GlobalHelp() string { return GlobalHelp }

// Actions returns the list of registered action names (unsorted).
func (r *Registry[C]) Actions() []string {
	names := make([]string, 0, len(r.commands))
	for k := range r.commands {
		names = append(names, k)
	}
	return names
}
