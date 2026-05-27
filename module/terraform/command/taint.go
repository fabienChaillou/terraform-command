package command

import (
	"strings"

	"github.com/fabienChaillou/terraform-commander/terraform"
)

const taintHelp = `Usage: terraform taint [options] ADDRESS

Mark a resource as tainted so the next apply will recreate it.

Options:
  -allow-missing       Succeed even if the resource does not exist.
  -lock=true|false     Acquire a state lock before changing state (default true).
  -lock-timeout=DUR    How long to wait for the lock (e.g. 30s, 5m).

Example:
  terraform taint aws_instance.web
`

// knownTaintFlags is the set of flag keys accepted by taint and untaint.
// Both commands share the same flag set in the modern Terraform CLI.
var knownTaintFlags = map[string]bool{
	"address":       true,
	"allow-missing": true,
	"lock":          true,
	"lock-timeout":  true,
}

// TaintCommand implements terraform.Command for "terraform taint".
type TaintCommand struct{}

func (c *TaintCommand) Help() string { return taintHelp }

func (c *TaintCommand) Validate(req *terraform.Request) []terraform.ValidationError {
	return validateTaintLike(req, "taint")
}

func (c *TaintCommand) BuildArgs(req *terraform.Request) []string {
	return buildTaintLikeArgs(req, "taint")
}

// validateTaintLike is shared by taint and untaint — they accept exactly the
// same flag set and require the same positional address.
func validateTaintLike(req *terraform.Request, cmdName string) []terraform.ValidationError {
	var errs []terraform.ValidationError

	if e := requireStringArg(req.Args, "address",
		`terraform `+cmdName+` requires an address: {"address": "<resource.address>"}`); e != nil {
		errs = append(errs, *e)
	}

	if v, ok := req.Args["lock"]; ok {
		switch v.(type) {
		case bool, string:
			// strings are accepted because JSON callers sometimes send "true" / "false".
		default:
			errs = append(errs, terraform.ValidationError{
				Field:   "lock",
				Message: "-lock must be a boolean (true | false)",
			})
		}
	}
	if v, ok := req.Args["lock-timeout"]; ok {
		if s, isStr := v.(string); !isStr || strings.TrimSpace(s) == "" {
			errs = append(errs, terraform.ValidationError{
				Field:   "lock-timeout",
				Message: "-lock-timeout must be a non-empty duration string (e.g. \"30s\", \"5m\")",
			})
		}
	}

	errs = append(errs, unknownFlagErrors(req.Args, knownTaintFlags, cmdName)...)
	return errs
}

// buildTaintLikeArgs is shared by taint and untaint.
// Output order: <verb> [-allow-missing] [-lock=BOOL] [-lock-timeout=DUR] ADDRESS
func buildTaintLikeArgs(req *terraform.Request, verb string) []string {
	args := []string{verb}

	if isTruthy(req.Args["allow-missing"]) {
		args = append(args, "-allow-missing")
	}
	// -lock is explicitly false-able (default true), so we emit only when present.
	if v, ok := req.Args["lock"]; ok {
		if isTruthy(v) {
			args = append(args, "-lock=true")
		} else {
			args = append(args, "-lock=false")
		}
	}
	if v, ok := req.Args["lock-timeout"].(string); ok && v != "" {
		args = append(args, "-lock-timeout="+v)
	}
	if addr, ok := req.Args["address"].(string); ok && addr != "" {
		args = append(args, addr)
	}
	return args
}
