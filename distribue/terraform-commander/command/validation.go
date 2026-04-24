package command

// Package-level validation helpers shared by all Command implementations.
//
// Keeping these in a dedicated file satisfies the Single Responsibility
// Principle: the registry manages command lookup; this file owns the
// reusable validation logic that every command delegates to.

import (
	"fmt"
	"strings"
)

// ── Flag validation ───────────────────────────────────────────────────────────

// unknownFlagErrors returns a ValidationError for every key in args that is
// not in the allowed set.  The comparison is case-insensitive on the key.
func unknownFlagErrors(args map[string]interface{}, allowed map[string]bool, cmdName string) []ValidationError {
	var errs []ValidationError
	for k := range args {
		if !allowed[strings.ToLower(k)] {
			errs = append(errs, ValidationError{
				Field:   k,
				Message: fmt.Sprintf("unknown flag for terraform %s: -%s", cmdName, k),
			})
		}
	}
	return errs
}

// requireStringArg returns a ValidationError if the given key is absent in
// args or its value is an empty / whitespace-only string.
func requireStringArg(args map[string]interface{}, key, hint string) *ValidationError {
	v, ok := args[key]
	if !ok || strings.TrimSpace(fmt.Sprint(v)) == "" {
		return &ValidationError{Field: key, Message: hint}
	}
	return nil
}

// ── Value coercion helpers ────────────────────────────────────────────────────

// isTruthy reports whether v represents a boolean true value.
// Accepts a Go bool, the string "true" (case-insensitive), or the string "1".
// Any other type (including nil) is considered false.
//
// This coercion is needed because JSON unmarshalling into
// map[string]interface{} always produces float64 for numbers and string for
// strings — callers may pass "true" instead of true.
func isTruthy(v interface{}) bool {
	switch b := v.(type) {
	case bool:
		return b
	case string:
		s := strings.ToLower(b)
		return s == "true" || s == "1"
	}
	return false
}
