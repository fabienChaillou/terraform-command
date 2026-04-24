package command

// ── Execution result ──────────────────────────────────────────────────────────

// ExecutionResult is the output produced by the worker after running the
// shell command.  It is the single shared type between the api layer
// (consumer) and the temporal layer (producer).
//
// A non-zero ExitCode is NOT a Go error: it is a valid terraform failure
// (e.g. plan diff exists, apply failed due to missing provider).
// Go errors from the activity layer signal infrastructure problems
// (binary not found, worker crash, timeout).
type ExecutionResult struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

// Success returns true if the command exited with code 0.
func (r *ExecutionResult) Success() bool { return r.ExitCode == 0 }

// ── Execution options ─────────────────────────────────────────────────────────

// ExecuteOptions carries per-call execution tuning parameters.
// Defined here (in the command package) so both api and temporal can reference
// the same type without creating a circular dependency.
type ExecuteOptions struct {
	// Action is the terraform sub-command name (init | plan | apply | destroy | workspace).
	// The executor uses it to select the appropriate workflow function via its
	// WorkflowByAction registry.  An empty string falls back to the default workflow.
	Action string

	// TimeoutSeconds is the maximum duration for the shell command.
	// 0 uses the executor's default.
	TimeoutSeconds int

	// MaxAttempts is the maximum number of execution attempts.
	// 0 or 1 = no retry.
	MaxAttempts int32
}

// ── Generic action map ────────────────────────────────────────────────────────

// ActionMap[T] is a generic map from action name to a typed configuration
// value.
//
// It exists to replace plain map[string]T in api.Config: a plain map silently
// returns the zero value when a key is missing, which can lead to subtle
// bugs (e.g. a 0-second timeout or 0 max-attempts).  ActionMap.Get requires
// callers to supply an explicit default, making the intent visible at the
// call site.
//
// # Usage
//
//	timeouts := command.ActionMap[int]{
//	    "apply":   1800,
//	    "plan":    900,
//	}
//	t := timeouts.Get("plan",  300) // → 900  (key present)
//	t  = timeouts.Get("init",  300) // → 300  (key absent, uses default)
//	t  = timeouts.Get("apply", 300) // → 1800 (key present)
type ActionMap[T any] map[string]T

// Get returns the value stored under action, or defaultVal if action is not
// present in the map.
func (m ActionMap[T]) Get(action string, defaultVal T) T {
	if v, ok := m[action]; ok {
		return v
	}
	return defaultVal
}

// Set stores v under action.  It is a convenience wrapper that initialises
// the underlying map when m is nil (as opposed to a plain map which would
// panic on write to a nil map).
func (m *ActionMap[T]) Set(action string, v T) {
	if *m == nil {
		*m = make(ActionMap[T])
	}
	(*m)[action] = v
}
