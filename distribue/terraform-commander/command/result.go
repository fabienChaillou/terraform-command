package command

import "github.com/fabienChaillou/terraform-contracts/contracts"

// ── Type aliases ──────────────────────────────────────────────────────────────
//
// ExecutionResult and ExecuteOptions are now owned by terraform-contracts so
// that both this repo (server) and terraform-commander-worker share the exact
// same type definitions with zero duplication.
//
// These aliases let existing code in this repo continue to reference
// command.ExecutionResult / command.ExecuteOptions without change.

// ExecutionResult is the output of a completed Terraform workflow.
// See contracts.ExecutionResult for the canonical definition.
type ExecutionResult = contracts.ExecutionResult

// ExecuteOptions carries per-call execution tuning for the Executor.
// See contracts.ExecuteOptions for the canonical definition.
type ExecuteOptions = contracts.ExecuteOptions

// ActionMap[T] is a generic map from action name to a typed configuration
// value.  See contracts.ActionMap for the canonical definition.
//
// NOTE: Go 1.22 does not support generic type aliases, so ActionMap is
// redefined here as a concrete wrapper type.  Use contracts.ActionMap directly
// in new code; this copy exists only to keep the existing command-package API
// stable until the module is updated to Go ≥ 1.24 or callers are migrated.
type ActionMap[T any] map[string]T

// Get returns the value stored under action, or defaultVal when absent.
func (m ActionMap[T]) Get(action string, defaultVal T) T {
	if v, ok := m[action]; ok {
		return v
	}
	return defaultVal
}

// Set stores v under action, initialising the map when m is nil.
func (m *ActionMap[T]) Set(action string, v T) {
	if *m == nil {
		*m = make(ActionMap[T])
	}
	(*m)[action] = v
}
