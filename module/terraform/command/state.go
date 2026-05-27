package command

import (
	"fmt"
	"strings"

	"github.com/fabienChaillou/terraform-commander/terraform"
)

const stateHelp = `Usage: terraform state <subcommand> [options] [args]

Advanced state management.

Subcommands:
  list                List resources in the state (optional address filter).
  show    <ADDRESS>   Show a resource's attributes.
  mv      <SRC> <DST> Move an item in the state.
  rm      <ADDRESS>   Remove an item from the state.
  pull                Download remote state and print to stdout.
  push    <PATH>      Upload a local state file to the configured backend.

Common options (per subcommand):
  -state=PATH         Path to read state from (list, show, mv, rm).
  -state-out=PATH     Path to write state to (mv).
  -dry-run            Print actions without changing state (mv).
  -force              Bypass safety checks (push).
`

var stateSubcmdHelp = map[string]string{
	"list": `Usage: terraform state list [options] [ADDRESS...]

List resources currently tracked in the state file.
Options:
  -state=PATH   Path to a custom state file.
  -id=VALUE     Filter resources matching the given id attribute.
`,
	"show": `Usage: terraform state show [options] ADDRESS

Show the attributes of a single resource.
Options:
  -state=PATH   Path to a custom state file.
`,
	"mv": `Usage: terraform state mv [options] SOURCE DESTINATION

Move an item in the Terraform state.
Options:
  -state=PATH       Path to the source state file.
  -state-out=PATH   Path to the destination state file (defaults to -state).
  -dry-run          Print the move without performing it.
`,
	"rm": `Usage: terraform state rm [options] ADDRESS

Remove one or more items from the Terraform state.
Options:
  -state=PATH   Path to a custom state file.
  -dry-run      Print actions without modifying state.
`,
	"pull": `Usage: terraform state pull

Pull the current state from the configured backend and write it to stdout.
`,
	"push": `Usage: terraform state push [options] PATH

Push a local state file to the configured backend.
Options:
  -force   Skip safety checks when the lineage or serial differs.
`,
}

var knownStateSubcmds = map[string]bool{
	"list": true, "show": true, "mv": true,
	"rm": true, "pull": true, "push": true,
}

// StateCommand implements terraform.Command for "terraform state".
type StateCommand struct{}

func (c *StateCommand) Help() string { return stateHelp }

func (c *StateCommand) helpFor(sub string) string {
	if h, ok := stateSubcmdHelp[strings.ToLower(sub)]; ok {
		return h
	}
	return stateHelp
}

func (c *StateCommand) Validate(req *terraform.Request) []terraform.ValidationError {
	sub := strings.ToLower(strings.TrimSpace(req.Subcommand))

	if sub == "" {
		return []terraform.ValidationError{{
			Field:   "subcommand",
			Message: "terraform state requires a subcommand: list | show | mv | rm | pull | push",
		}}
	}
	if !knownStateSubcmds[sub] {
		return []terraform.ValidationError{{
			Field:   "subcommand",
			Message: fmt.Sprintf("unknown state subcommand %q — valid: list | show | mv | rm | pull | push", sub),
		}}
	}

	switch sub {
	case "list":
		return validateStateList(req)
	case "show":
		return validateStateShow(req)
	case "mv":
		return validateStateMv(req)
	case "rm":
		return validateStateRm(req)
	case "pull":
		return validateNoArgs(req, "state pull")
	case "push":
		return validateStatePush(req)
	}
	return nil
}

// BuildArgs produces the flat CLI args for the state subcommand.
// Positional arguments are appended after flags, in the order documented
// by terraform's CLI: e.g. `state mv [options] SOURCE DEST`.
func (c *StateCommand) BuildArgs(req *terraform.Request) []string {
	sub := strings.ToLower(req.Subcommand)
	base := []string{"state", sub}

	switch sub {
	case "list":
		if state, ok := req.Args["state"].(string); ok && state != "" {
			base = append(base, "-state="+state)
		}
		if id, ok := req.Args["id"].(string); ok && id != "" {
			base = append(base, "-id="+id)
		}
		// Optional positional address(es) — accept either a single string
		// or a []string / []interface{}.
		base = append(base, positionalAddresses(req.Args["address"])...)

	case "show":
		if state, ok := req.Args["state"].(string); ok && state != "" {
			base = append(base, "-state="+state)
		}
		if addr, ok := req.Args["address"].(string); ok {
			base = append(base, addr)
		}

	case "mv":
		if state, ok := req.Args["state"].(string); ok && state != "" {
			base = append(base, "-state="+state)
		}
		if out, ok := req.Args["state-out"].(string); ok && out != "" {
			base = append(base, "-state-out="+out)
		}
		if isTruthy(req.Args["dry-run"]) {
			base = append(base, "-dry-run")
		}
		if src, ok := req.Args["source"].(string); ok {
			base = append(base, src)
		}
		if dst, ok := req.Args["destination"].(string); ok {
			base = append(base, dst)
		}

	case "rm":
		if state, ok := req.Args["state"].(string); ok && state != "" {
			base = append(base, "-state="+state)
		}
		if isTruthy(req.Args["dry-run"]) {
			base = append(base, "-dry-run")
		}
		base = append(base, positionalAddresses(req.Args["address"])...)

	case "push":
		if isTruthy(req.Args["force"]) {
			base = append(base, "-force")
		}
		if path, ok := req.Args["path"].(string); ok {
			base = append(base, path)
		}

		// pull: no flags / no positional args
	}
	return base
}

// ── per-subcommand validators ─────────────────────────────────────────────────

func validateStateList(req *terraform.Request) []terraform.ValidationError {
	var errs []terraform.ValidationError

	if v, ok := req.Args["state"]; ok {
		if s, isStr := v.(string); !isStr || strings.TrimSpace(s) == "" {
			errs = append(errs, terraform.ValidationError{Field: "state", Message: "-state must be a non-empty file path"})
		}
	}
	if v, ok := req.Args["id"]; ok {
		if s, isStr := v.(string); !isStr || strings.TrimSpace(s) == "" {
			errs = append(errs, terraform.ValidationError{Field: "id", Message: "-id must be a non-empty string"})
		}
	}
	if v, ok := req.Args["address"]; ok {
		// Accept string, []string, []interface{} of strings.
		if !isAddressValueOK(v) {
			errs = append(errs, terraform.ValidationError{
				Field: "address", Message: "address must be a string or array of strings",
			})
		}
	}
	known := map[string]bool{"state": true, "id": true, "address": true}
	errs = append(errs, unknownFlagErrors(req.Args, known, "state list")...)
	return errs
}

func validateStateShow(req *terraform.Request) []terraform.ValidationError {
	var errs []terraform.ValidationError
	if e := requireStringArg(req.Args, "address",
		`state show requires an address: {"address": "<resource.address>"}`); e != nil {
		errs = append(errs, *e)
	}
	if v, ok := req.Args["state"]; ok {
		if s, isStr := v.(string); !isStr || strings.TrimSpace(s) == "" {
			errs = append(errs, terraform.ValidationError{Field: "state", Message: "-state must be a non-empty file path"})
		}
	}
	known := map[string]bool{"address": true, "state": true}
	errs = append(errs, unknownFlagErrors(req.Args, known, "state show")...)
	return errs
}

func validateStateMv(req *terraform.Request) []terraform.ValidationError {
	var errs []terraform.ValidationError
	if e := requireStringArg(req.Args, "source",
		`state mv requires a source: {"source": "<addr>"}`); e != nil {
		errs = append(errs, *e)
	}
	if e := requireStringArg(req.Args, "destination",
		`state mv requires a destination: {"destination": "<addr>"}`); e != nil {
		errs = append(errs, *e)
	}
	for _, key := range []string{"state", "state-out"} {
		if v, ok := req.Args[key]; ok {
			if s, isStr := v.(string); !isStr || strings.TrimSpace(s) == "" {
				errs = append(errs, terraform.ValidationError{Field: key, Message: "-" + key + " must be a non-empty file path"})
			}
		}
	}
	known := map[string]bool{"source": true, "destination": true, "state": true, "state-out": true, "dry-run": true}
	errs = append(errs, unknownFlagErrors(req.Args, known, "state mv")...)
	return errs
}

func validateStateRm(req *terraform.Request) []terraform.ValidationError {
	var errs []terraform.ValidationError
	v, ok := req.Args["address"]
	if !ok || !isAddressValueOK(v) {
		errs = append(errs, terraform.ValidationError{
			Field: "address", Message: `state rm requires an address: {"address": "<addr>"}`,
		})
	}
	if v, ok := req.Args["state"]; ok {
		if s, isStr := v.(string); !isStr || strings.TrimSpace(s) == "" {
			errs = append(errs, terraform.ValidationError{Field: "state", Message: "-state must be a non-empty file path"})
		}
	}
	known := map[string]bool{"address": true, "state": true, "dry-run": true}
	errs = append(errs, unknownFlagErrors(req.Args, known, "state rm")...)
	return errs
}

func validateStatePush(req *terraform.Request) []terraform.ValidationError {
	var errs []terraform.ValidationError
	if e := requireStringArg(req.Args, "path",
		`state push requires a path: {"path": "<file>"}`); e != nil {
		errs = append(errs, *e)
	}
	known := map[string]bool{"path": true, "force": true}
	errs = append(errs, unknownFlagErrors(req.Args, known, "state push")...)
	return errs
}

// ── value helpers (state-specific) ───────────────────────────────────────────

// isAddressValueOK accepts a single string or a slice of strings; non-empty.
func isAddressValueOK(v interface{}) bool {
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x) != ""
	case []string:
		if len(x) == 0 {
			return false
		}
		for _, s := range x {
			if strings.TrimSpace(s) == "" {
				return false
			}
		}
		return true
	case []interface{}:
		if len(x) == 0 {
			return false
		}
		for _, e := range x {
			s, isStr := e.(string)
			if !isStr || strings.TrimSpace(s) == "" {
				return false
			}
		}
		return true
	}
	return false
}

// positionalAddresses flattens v into a slice of address strings.
// Returns nil if v is nil or empty.
func positionalAddresses(v interface{}) []string {
	switch x := v.(type) {
	case string:
		if strings.TrimSpace(x) == "" {
			return nil
		}
		return []string{x}
	case []string:
		return append([]string(nil), x...)
	case []interface{}:
		out := make([]string, 0, len(x))
		for _, e := range x {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}
