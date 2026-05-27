package command

import (
	"fmt"
	"strings"

	"github.com/fabienChaillou/terraform-commander/terraform"
)

const workspaceHelp = `Usage: terraform workspace <subcommand> [options] [args]

Manage Terraform workspaces.

Subcommands:
  new     <name>   Create a new workspace
  select  <name>   Switch to a workspace
  list             List all workspaces
  delete  <name>   Delete a workspace
  show             Show the current workspace

Use "terraform workspace <subcommand> --help" for details.
`

var workspaceSubcmdHelp = map[string]string{
	"new": `Usage: terraform workspace new [options] NAME

Create a new workspace.

Options:
  -state=path   Copy an existing state file into the new workspace.

Arguments:
  name   Workspace name (required, no spaces).
`,
	"select": `Usage: terraform workspace select NAME

Switch to the named workspace.
`,
	"list": `Usage: terraform workspace list

List all workspaces. The current workspace is marked with *.
`,
	"delete": `Usage: terraform workspace delete [options] NAME

Delete a workspace.

Options:
  -force   Remove a non-empty workspace.
`,
	"show": `Usage: terraform workspace show

Display the name of the current workspace.
`,
}

var knownWorkspaceSubcmds = map[string]bool{
	"new": true, "select": true, "list": true, "delete": true, "show": true,
}

// WorkspaceCommand implements terraform.Command for "terraform workspace".
type WorkspaceCommand struct{}

func (c *WorkspaceCommand) Help() string { return workspaceHelp }

func (c *WorkspaceCommand) helpFor(sub string) string {
	if h, ok := workspaceSubcmdHelp[strings.ToLower(sub)]; ok {
		return h
	}
	return workspaceHelp
}

func (c *WorkspaceCommand) Validate(req *terraform.Request) []terraform.ValidationError {
	sub := strings.ToLower(strings.TrimSpace(req.Subcommand))

	// Missing subcommand.
	if sub == "" {
		return []terraform.ValidationError{{
			Field:   "subcommand",
			Message: "terraform workspace requires a subcommand: new | select | list | delete | show",
		}}
	}

	// Unknown subcommand.
	if !knownWorkspaceSubcmds[sub] {
		return []terraform.ValidationError{{
			Field:   "subcommand",
			Message: fmt.Sprintf("unknown workspace subcommand %q — valid: new | select | list | delete | show", sub),
		}}
	}

	switch sub {
	case "new":
		return validateWorkspaceNew(req)
	case "select":
		return validateWorkspaceSelect(req)
	case "delete":
		return validateWorkspaceDelete(req)
	case "list", "show":
		return validateNoArgs(req, "workspace "+sub)
	}
	return nil
}

// BuildArgs produces the flat CLI args for the workspace subcommand.
// Positional arguments (workspace name) are appended after flags.
func (c *WorkspaceCommand) BuildArgs(req *terraform.Request) []string {
	sub := strings.ToLower(req.Subcommand)
	base := []string{"workspace", sub}

	switch sub {
	case "new":
		if state, ok := req.Args["state"].(string); ok {
			base = append(base, "-state="+state)
		}
		if name, ok := req.Args["name"].(string); ok {
			base = append(base, name)
		}

	case "select":
		if name, ok := req.Args["name"].(string); ok {
			base = append(base, name)
		}

	case "delete":
		if isTruthy(req.Args["force"]) {
			base = append(base, "-force")
		}
		if name, ok := req.Args["name"].(string); ok {
			base = append(base, name)
		}

		// list / show: no additional args
	}

	return base
}

// ── per-subcommand validators ─────────────────────────────────────────────────

func validateWorkspaceNew(req *terraform.Request) []terraform.ValidationError {
	var errs []terraform.ValidationError

	if e := requireStringArg(req.Args, "name",
		`workspace new requires a name: {"name": "<workspace>"}`); e != nil {
		errs = append(errs, *e)
	} else if name := fmt.Sprint(req.Args["name"]); strings.ContainsAny(name, " \t") {
		errs = append(errs, terraform.ValidationError{Field: "name", Message: "workspace name must not contain spaces"})
	}

	if v, ok := req.Args["state"]; ok {
		if s, isStr := v.(string); !isStr || strings.TrimSpace(s) == "" {
			errs = append(errs, terraform.ValidationError{Field: "state", Message: "-state must be a non-empty file path"})
		}
	}

	known := map[string]bool{"name": true, "state": true}
	errs = append(errs, unknownFlagErrors(req.Args, known, "workspace new")...)
	return errs
}

func validateWorkspaceSelect(req *terraform.Request) []terraform.ValidationError {
	var errs []terraform.ValidationError
	if e := requireStringArg(req.Args, "name",
		`workspace select requires a name: {"name": "<workspace>"}`); e != nil {
		errs = append(errs, *e)
	}
	errs = append(errs, unknownFlagErrors(req.Args, map[string]bool{"name": true}, "workspace select")...)
	return errs
}

func validateWorkspaceDelete(req *terraform.Request) []terraform.ValidationError {
	var errs []terraform.ValidationError
	if e := requireStringArg(req.Args, "name",
		`workspace delete requires a name: {"name": "<workspace>"}`); e != nil {
		errs = append(errs, *e)
	}
	errs = append(errs, unknownFlagErrors(req.Args, map[string]bool{"name": true, "force": true}, "workspace delete")...)
	return errs
}

func validateNoArgs(req *terraform.Request, cmdLabel string) []terraform.ValidationError {
	var errs []terraform.ValidationError
	for k := range req.Args {
		errs = append(errs, terraform.ValidationError{
			Field:   k,
			Message: fmt.Sprintf("terraform %s does not accept arguments", cmdLabel),
		})
	}
	return errs
}
