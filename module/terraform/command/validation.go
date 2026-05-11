// Package command provides the concrete Terraform sub-command implementations
// (init, plan, apply, destroy, workspace) and a preloaded registry constructor.
//
// The Command interface, Request, ValidationError, the generic Registry[C],
// ExecutionResult, ExecuteOptions and ActionMap[T] all live in the parent
// package github.com/fabienChaillou/terraform-commander/internal/terraform.
//
// Dependency direction:
//
//	internal/terraform/command  ──imports──►  internal/terraform
//
// The parent package never imports back, so there is no cycle.
package command

import (
	"fmt"
	"strings"

	"github.com/fabienChaillou/terraform-commander/internal/terraform"
)

// validation.go owns the reusable validation helpers shared by every
// concrete command in this package.  They are unexported because they are
// implementation details — callers should rely on each command's Validate
// method, never on these helpers directly.

// ── Flag validation ───────────────────────────────────────────────────────────

// unknownFlagErrors returns a ValidationError for every key in args that is
// not in the allowed set.  The comparison is case-insensitive on the key.
func unknownFlagErrors(args map[string]interface{}, allowed map[string]bool, cmdName string) []terraform.ValidationError {
	var errs []terraform.ValidationError
	for k := range args {
		if !allowed[strings.ToLower(k)] {
			errs = append(errs, terraform.ValidationError{
				Field:   k,
				Message: fmt.Sprintf("unknown flag for terraform %s: -%s", cmdName, k),
			})
		}
	}
	return errs
}

// requireStringArg returns a ValidationError if the given key is absent in
// args or its value is an empty / whitespace-only string.
func requireStringArg(args map[string]interface{}, key, hint string) *terraform.ValidationError {
	v, ok := args[key]
	if !ok || strings.TrimSpace(fmt.Sprint(v)) == "" {
		return &terraform.ValidationError{Field: key, Message: hint}
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
