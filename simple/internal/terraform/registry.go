package terraform

import "fmt"

// Registry holds the whitelist of all known Terraform commands
type Registry struct {
	commands map[string]Command
}

// NewRegistry creates a Registry pre-loaded with all supported Terraform commands
func NewRegistry() *Registry {
	r := &Registry{commands: make(map[string]Command)}

	cmds := []Command{
		NewInitCommand(),
		NewValidateCommand(),
		NewPlanCommand(),
		NewApplyCommand(),
		NewDestroyCommand(),
		NewStateCommand(),
		NewWorkspaceCommand(),
		NewOutputCommand(),
		NewImportCommand(),
		NewShowCommand(),
		NewFmtCommand(),
		NewRefreshCommand(),
		NewProvidersCommand(),
		NewGraphCommand(),
	}

	for _, cmd := range cmds {
		r.commands[cmd.Name()] = cmd
	}
	return r
}

// Lookup returns the Command for a given action name (whitelist check)
func (r *Registry) Lookup(action string) (Command, bool) {
	cmd, ok := r.commands[action]
	return cmd, ok
}

// Actions returns a sorted list of all registered command names
func (r *Registry) Actions() []string {
	names := make([]string, 0, len(r.commands))
	for name := range r.commands {
		names = append(names, name)
	}
	return names
}

// ─── DISPATCHER ───────────────────────────────────────────────────────────────

// Dispatcher processes a raw API payload through the full pipeline:
//  1. Whitelist check   → lookup the Command
//  2. Validate inputs   → call Command.Validate()
//  3. Return Help       → on any validation error
//  4. Return result     → clean, validated payload
type Dispatcher struct {
	registry *Registry
}

func NewDispatcher(r *Registry) *Dispatcher { return &Dispatcher{registry: r} }

// Dispatch executes the full pipeline and returns a CommandResult.
// It never returns a Go error; errors are surfaced inside CommandResult.
func (d *Dispatcher) Dispatch(payload map[string]any) CommandResult {
	// ── 1. Extract action ───────────────────────────────────────────────────
	actionRaw, ok := payload["action"]
	if !ok {
		return CommandResult{
			Valid:    false,
			Errors:   []ValidationError{{Field: "action", Message: "field 'action' is required"}},
			HelpText: d.globalHelp(),
		}
	}
	action, ok := actionRaw.(string)
	if !ok || action == "" {
		return CommandResult{
			Valid:    false,
			Errors:   []ValidationError{{Field: "action", Message: "field 'action' must be a non-empty string"}},
			HelpText: d.globalHelp(),
		}
	}

	// ── 2. Whitelist check ──────────────────────────────────────────────────
	cmd, found := d.registry.Lookup(action)
	if !found {
		return CommandResult{
			Valid: false,
			Errors: []ValidationError{{
				Field:   "action",
				Message: fmt.Sprintf("unknown action %q — allowed: %v", action, d.registry.Actions()),
			}},
			HelpText: d.globalHelp(),
		}
	}

	// Strip the action key before forwarding
	params := copyWithout(payload, "action")

	// ── 3. Validate ─────────────────────────────────────────────────────────
	clean, errs := cmd.Validate(params)

	if len(errs) > 0 {
		// ── 4. Return Help on errors ─────────────────────────────────────────
		return CommandResult{
			Command:  action,
			Valid:    false,
			Errors:   errs,
			HelpText: cmd.Help(),
		}
	}

	// ── 5. Success ───────────────────────────────────────────────────────────
	result := CommandResult{
		Command: action,
		Valid:   true,
		Payload: clean,
	}

	// Attach sub_command metadata if relevant
	if sub, ok := clean["sub_command"].(string); ok {
		result.SubCommand = sub
	}

	return result
}

func (d *Dispatcher) globalHelp() string {
	return fmt.Sprintf("Available actions: %v", d.registry.Actions())
}

func copyWithout(m map[string]any, key string) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		if k != key {
			out[k] = v
		}
	}
	return out
}
