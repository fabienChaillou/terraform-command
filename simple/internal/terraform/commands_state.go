package terraform

// ─── STATE ────────────────────────────────────────────────────────────────────

type StateCommand struct {
	BaseCommand
	subCommands []string
}

func NewStateCommand() *StateCommand {
	return &StateCommand{
		BaseCommand: BaseCommand{"state"},
		subCommands: []string{"list", "show", "mv", "rm", "pull", "push", "replace-provider"},
	}
}

func (c *StateCommand) SubCommands() []string { return c.subCommands }

func (c *StateCommand) RunSub(sub string, args []string) error {
	_ = sub
	return c.Run(args)
}

func (c *StateCommand) Help() string {
	return `Usage: terraform state <subcommand> [options]

Manage Terraform state.

Subcommands:
  list              List resources in state.
  show              Show a resource in state.
  mv                Move a resource in state.
  rm                Remove a resource from state.
  pull              Pull state from backend.
  push              Push state to backend.
  replace-provider  Replace provider in state.

Options:
  sub_command (string, required)  One of the subcommands above.
  address  (string)               Resource address (required for show/mv/rm).
  destination (string)            Destination address (required for mv).`
}

func (c *StateCommand) Validate(payload map[string]any) (map[string]any, []ValidationError) {
	var errs []ValidationError
	clean := map[string]any{}

	sub, ok := getString(payload, "sub_command")
	if !ok || sub == "" {
		errs = append(errs, ValidationError{"sub_command", "required: one of list|show|mv|rm|pull|push|replace-provider"})
		return clean, errs
	}

	valid := false
	for _, s := range c.subCommands {
		if s == sub {
			valid = true
			break
		}
	}
	if !valid {
		errs = append(errs, ValidationError{"sub_command", "unknown subcommand: " + sub})
		return clean, errs
	}
	clean["sub_command"] = sub

	switch sub {
	case "show", "rm":
		addr, ok := getString(payload, "address")
		if !ok || addr == "" {
			errs = append(errs, ValidationError{"address", "required for sub_command=" + sub})
		} else {
			clean["address"] = addr
		}
	case "mv":
		addr, ok := getString(payload, "address")
		if !ok || addr == "" {
			errs = append(errs, ValidationError{"address", "required for sub_command=mv"})
		} else {
			clean["address"] = addr
		}
		dest, ok := getString(payload, "destination")
		if !ok || dest == "" {
			errs = append(errs, ValidationError{"destination", "required for sub_command=mv"})
		} else {
			clean["destination"] = dest
		}
	}

	return clean, errs
}

// ─── OUTPUT ───────────────────────────────────────────────────────────────────

type OutputCommand struct{ BaseCommand }

func NewOutputCommand() *OutputCommand { return &OutputCommand{BaseCommand{"output"}} }

func (c *OutputCommand) Help() string {
	return `Usage: terraform output [options] [NAME]

Show output values from your root module.

Options:
  name    (string) Specific output name to show.
  -json           Format output as JSON.
  -raw            Print raw string value.
  -no-color       Disable color output.`
}

func (c *OutputCommand) Validate(payload map[string]any) (map[string]any, []ValidationError) {
	clean := map[string]any{}

	if name, ok := getString(payload, "name"); ok && name != "" {
		clean["name"] = name
	}
	if jsonOut, ok := getBool(payload, "json"); ok {
		clean["json"] = jsonOut
	}
	if raw, ok := getBool(payload, "raw"); ok {
		clean["raw"] = raw
	}
	return clean, nil
}

// ─── IMPORT ───────────────────────────────────────────────────────────────────

type ImportCommand struct{ BaseCommand }

func NewImportCommand() *ImportCommand { return &ImportCommand{BaseCommand{"import"}} }

func (c *ImportCommand) Help() string {
	return `Usage: terraform import [options] ADDR ID

Import existing infrastructure into Terraform state.

Required:
  address (string) Resource address (e.g., aws_instance.example).
  id      (string) Provider-specific resource ID.

Options:
  -var=k=v         Set a variable value.
  -var-file=path   Load variable values from a file.
  -allow-missing-config Allow import without configuration.`
}

func (c *ImportCommand) Validate(payload map[string]any) (map[string]any, []ValidationError) {
	var errs []ValidationError
	clean := map[string]any{}

	addr, ok := getString(payload, "address")
	if !ok || addr == "" {
		errs = append(errs, ValidationError{"address", "required (e.g., aws_instance.example)"})
	} else {
		clean["address"] = addr
	}

	id, ok := getString(payload, "id")
	if !ok || id == "" {
		errs = append(errs, ValidationError{"id", "required: provider-specific resource ID"})
	} else {
		clean["id"] = id
	}

	if vars, ok := getStringSlice(payload, "var"); ok {
		clean["var"] = vars
	}
	if varFile, ok := getString(payload, "var_file"); ok && varFile != "" {
		clean["var_file"] = varFile
	}
	return clean, errs
}

// ─── SHOW ─────────────────────────────────────────────────────────────────────

type ShowCommand struct{ BaseCommand }

func NewShowCommand() *ShowCommand { return &ShowCommand{BaseCommand{"show"}} }

func (c *ShowCommand) Help() string {
	return `Usage: terraform show [options] [path]

Show the current state or a saved plan.

Options:
  path   (string) Path to a plan file (optional).
  -json          Produce JSON output.
  -no-color      Disable color.`
}

func (c *ShowCommand) Validate(payload map[string]any) (map[string]any, []ValidationError) {
	clean := map[string]any{}

	if path, ok := getString(payload, "path"); ok && path != "" {
		clean["path"] = path
	}
	if jsonOut, ok := getBool(payload, "json"); ok {
		clean["json"] = jsonOut
	}
	return clean, nil
}

// ─── FMT ──────────────────────────────────────────────────────────────────────

type FmtCommand struct{ BaseCommand }

func NewFmtCommand() *FmtCommand { return &FmtCommand{BaseCommand{"fmt"}} }

func (c *FmtCommand) Help() string {
	return `Usage: terraform fmt [options] [dir]

Reformat configuration files to canonical format.

Options:
  dir      (string) Directory to format (default: current).
  -check           Check if files are formatted, exit non-zero if not.
  -diff            Display diffs of formatting changes.
  -recursive       Process files in subdirectories.`
}

func (c *FmtCommand) Validate(payload map[string]any) (map[string]any, []ValidationError) {
	clean := map[string]any{}

	if dir, ok := getString(payload, "dir"); ok && dir != "" {
		clean["dir"] = dir
	}
	if check, ok := getBool(payload, "check"); ok {
		clean["check"] = check
	}
	if diff, ok := getBool(payload, "diff"); ok {
		clean["diff"] = diff
	}
	if recursive, ok := getBool(payload, "recursive"); ok {
		clean["recursive"] = recursive
	}
	return clean, nil
}

// ─── REFRESH ──────────────────────────────────────────────────────────────────

type RefreshCommand struct{ BaseCommand }

func NewRefreshCommand() *RefreshCommand { return &RefreshCommand{BaseCommand{"refresh"}} }

func (c *RefreshCommand) Help() string {
	return `Usage: terraform refresh [options]

Update the state file to match real infrastructure.

Options:
  -var=k=v         Set a variable value.
  -var-file=path   Load variable values from a file.
  -target=resource Limit scope to a specific resource.
  -parallelism=n   Number of parallel resource operations.`
}

func (c *RefreshCommand) Validate(payload map[string]any) (map[string]any, []ValidationError) {
	clean := map[string]any{}

	if targets, ok := getStringSlice(payload, "target"); ok {
		clean["target"] = targets
	}
	if vars, ok := getStringSlice(payload, "var"); ok {
		clean["var"] = vars
	}
	if varFile, ok := getString(payload, "var_file"); ok && varFile != "" {
		clean["var_file"] = varFile
	}
	return clean, nil
}

// ─── PROVIDERS ────────────────────────────────────────────────────────────────

type ProvidersCommand struct{ BaseCommand }

func NewProvidersCommand() *ProvidersCommand { return &ProvidersCommand{BaseCommand{"providers"}} }

func (c *ProvidersCommand) Help() string {
	return `Usage: terraform providers [dir]

Show the providers required for this configuration.

Options:
  dir      (string) Working directory.
  -json           Output in JSON format.`
}

func (c *ProvidersCommand) Validate(payload map[string]any) (map[string]any, []ValidationError) {
	clean := map[string]any{}

	if dir, ok := getString(payload, "dir"); ok && dir != "" {
		clean["dir"] = dir
	}
	if jsonOut, ok := getBool(payload, "json"); ok {
		clean["json"] = jsonOut
	}
	return clean, nil
}

// ─── GRAPH ────────────────────────────────────────────────────────────────────

type GraphCommand struct{ BaseCommand }

func NewGraphCommand() *GraphCommand { return &GraphCommand{BaseCommand{"graph"}} }

func (c *GraphCommand) Help() string {
	return `Usage: terraform graph [options]

Generate a visual representation of the dependency graph.

Options:
  -type=string     Type of graph (plan, plan-refresh-only, plan-destroy, apply).
  -draw-cycles     Highlight cycles in the dependency graph.
  -plan=path       Render graph from a saved plan file.`
}

func (c *GraphCommand) Validate(payload map[string]any) (map[string]any, []ValidationError) {
	var errs []ValidationError
	clean := map[string]any{}

	validTypes := map[string]bool{
		"plan": true, "plan-refresh-only": true, "plan-destroy": true, "apply": true,
	}
	if t, ok := getString(payload, "type"); ok && t != "" {
		if !validTypes[t] {
			errs = append(errs, ValidationError{"type", "must be one of plan|plan-refresh-only|plan-destroy|apply"})
		} else {
			clean["type"] = t
		}
	}
	if drawCycles, ok := getBool(payload, "draw_cycles"); ok {
		clean["draw_cycles"] = drawCycles
	}
	return clean, errs
}
