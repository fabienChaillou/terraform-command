package terraform

// ─── WORKSPACE ────────────────────────────────────────────────────────────────

type WorkspaceCommand struct {
	BaseCommand
	subCommands []string
}

func NewWorkspaceCommand() *WorkspaceCommand {
	return &WorkspaceCommand{
		BaseCommand: BaseCommand{"workspace"},
		subCommands: []string{"list", "show", "new", "select", "delete"},
	}
}

func (c *WorkspaceCommand) SubCommands() []string { return c.subCommands }

func (c *WorkspaceCommand) RunSub(sub string, args []string) error {
	return c.Run(args)
}

func (c *WorkspaceCommand) Help() string {
	return `Usage: terraform workspace <subcommand> [options]

Manage Terraform workspaces.

Subcommands:
  list    List available workspaces.
  show    Show the current workspace name.
  new     Create a new workspace.
  select  Switch to a different workspace.
  delete  Delete a workspace.

Options:
  sub_command (string, required)  One of the subcommands above.
  name        (string)            Workspace name (required for new/select/delete).
  -force                          Force deletion even if not empty (delete only).
  -lock=true                      Lock state during workspace switch.
  -state=path                     Copy existing state file to new workspace (new only).`
}

func (c *WorkspaceCommand) Validate(payload map[string]any) (map[string]any, []ValidationError) {
	var errs []ValidationError
	clean := map[string]any{}

	sub, ok := getString(payload, "sub_command")
	if !ok || sub == "" {
		errs = append(errs, ValidationError{"sub_command", "required: one of list|show|new|select|delete"})
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
	case "new", "select", "delete":
		name, ok := getString(payload, "name")
		if !ok || name == "" {
			errs = append(errs, ValidationError{"name", "workspace name required for sub_command=" + sub})
		} else {
			clean["name"] = name
		}

		if sub == "delete" {
			if force, ok := getBool(payload, "force"); ok {
				clean["force"] = force
			}
		}
		if sub == "new" {
			if statePath, ok := getString(payload, "state"); ok && statePath != "" {
				clean["state"] = statePath
			}
		}
		if lock, ok := getBool(payload, "lock"); ok {
			clean["lock"] = lock
		}
	}

	return clean, errs
}
