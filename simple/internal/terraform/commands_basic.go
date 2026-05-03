package terraform

import "fmt"

// ─── INIT ────────────────────────────────────────────────────────────────────

type InitCommand struct{ BaseCommand }

func NewInitCommand() *InitCommand { return &InitCommand{BaseCommand{"init"}} }

func (c *InitCommand) Help() string {
	return `Usage: terraform init [options]

Initialize a Terraform working directory.

Options:
  -backend=true          Configure the backend for this configuration.
  -backend-config=path   Additional backend configuration.
  -upgrade               Upgrade modules and plugins to latest.
  -reconfigure           Reconfigure the backend.
  -migrate-state         Reconfigure backend and migrate state.
  dir (string)           Working directory (default: current).`
}

func (c *InitCommand) Validate(payload map[string]any) (map[string]any, []ValidationError) {
	var errs []ValidationError
	clean := map[string]any{}

	if dir, ok := getString(payload, "dir"); ok && dir != "" {
		clean["dir"] = dir
	}
	if backend, ok := getBool(payload, "backend"); ok {
		clean["backend"] = backend
	}
	if upgrade, ok := getBool(payload, "upgrade"); ok {
		clean["upgrade"] = upgrade
	}
	if reconfigure, ok := getBool(payload, "reconfigure"); ok {
		clean["reconfigure"] = reconfigure
	}
	if migrateState, ok := getBool(payload, "migrate_state"); ok {
		clean["migrate_state"] = migrateState
	}
	if backendConfig, ok := getString(payload, "backend_config"); ok && backendConfig != "" {
		clean["backend_config"] = backendConfig
	}

	return clean, errs
}

// ─── VALIDATE ────────────────────────────────────────────────────────────────

type ValidateCommand struct{ BaseCommand }

func NewValidateCommand() *ValidateCommand { return &ValidateCommand{BaseCommand{"validate"}} }

func (c *ValidateCommand) Help() string {
	return `Usage: terraform validate [options]

Validate the configuration files in a directory.

Options:
  -json   Produce output in machine-readable JSON format.
  -no-color Disable terminal color codes.
  dir (string) Working directory.`
}

func (c *ValidateCommand) Validate(payload map[string]any) (map[string]any, []ValidationError) {
	var errs []ValidationError
	clean := map[string]any{}

	if jsonOut, ok := getBool(payload, "json"); ok {
		clean["json"] = jsonOut
	}
	if noColor, ok := getBool(payload, "no_color"); ok {
		clean["no_color"] = noColor
	}
	if dir, ok := getString(payload, "dir"); ok && dir != "" {
		clean["dir"] = dir
	}
	return clean, errs
}

// ─── PLAN ─────────────────────────────────────────────────────────────────────

type PlanCommand struct{ BaseCommand }

func NewPlanCommand() *PlanCommand { return &PlanCommand{BaseCommand{"plan"}} }

func (c *PlanCommand) Help() string {
	return `Usage: terraform plan [options]

Show changes required by the current configuration.

Options:
  -out=path        Write the plan to a file.
  -var=k=v         Set a variable value.
  -var-file=path   Load variable values from a file.
  -destroy         Plan a destroy instead of an apply.
  -target=resource Limit scope to a specific resource.
  -refresh=true    Refresh state before planning.
  -compact-warnings Show compact warning output.`
}

func (c *PlanCommand) Validate(payload map[string]any) (map[string]any, []ValidationError) {
	var errs []ValidationError
	clean := map[string]any{}

	if out, ok := getString(payload, "out"); ok && out != "" {
		clean["out"] = out
	}
	if destroy, ok := getBool(payload, "destroy"); ok {
		clean["destroy"] = destroy
	}
	if refresh, ok := getBool(payload, "refresh"); ok {
		clean["refresh"] = refresh
	}
	if targets, ok := getStringSlice(payload, "target"); ok {
		clean["target"] = targets
	}
	if vars, ok := getStringSlice(payload, "var"); ok {
		clean["var"] = vars
	}
	if varFile, ok := getString(payload, "var_file"); ok && varFile != "" {
		clean["var_file"] = varFile
	}
	return clean, errs
}

// ─── APPLY ────────────────────────────────────────────────────────────────────

type ApplyCommand struct{ BaseCommand }

func NewApplyCommand() *ApplyCommand { return &ApplyCommand{BaseCommand{"apply"}} }

func (c *ApplyCommand) Help() string {
	return `Usage: terraform apply [options] [plan-file]

Apply the changes required to reach the desired state.

Options:
  -auto-approve    Skip interactive approval.
  -plan=path       Path to a pre-computed plan file.
  -var=k=v         Set a variable value.
  -var-file=path   Load variable values from a file.
  -target=resource Limit scope to a specific resource.
  -parallelism=n   Number of parallel resource operations (default 10).`
}

func (c *ApplyCommand) Validate(payload map[string]any) (map[string]any, []ValidationError) {
	var errs []ValidationError
	clean := map[string]any{}

	if autoApprove, ok := getBool(payload, "auto_approve"); ok {
		clean["auto_approve"] = autoApprove
	}
	if plan, ok := getString(payload, "plan"); ok && plan != "" {
		clean["plan"] = plan
	}
	if targets, ok := getStringSlice(payload, "target"); ok {
		clean["target"] = targets
	}
	if vars, ok := getStringSlice(payload, "var"); ok {
		clean["var"] = vars
	}
	if varFile, ok := getString(payload, "var_file"); ok && varFile != "" {
		clean["var_file"] = varFile
	}
	if p, ok := payload["parallelism"]; ok {
		switch v := p.(type) {
		case int:
			if v < 1 || v > 100 {
				errs = append(errs, ValidationError{"parallelism", "must be between 1 and 100"})
			} else {
				clean["parallelism"] = v
			}
		case float64:
			iv := int(v)
			if iv < 1 || iv > 100 {
				errs = append(errs, ValidationError{"parallelism", "must be between 1 and 100"})
			} else {
				clean["parallelism"] = iv
			}
		default:
			errs = append(errs, ValidationError{"parallelism", "must be an integer"})
		}
	}
	return clean, errs
}

// ─── DESTROY ──────────────────────────────────────────────────────────────────

type DestroyCommand struct{ BaseCommand }

func NewDestroyCommand() *DestroyCommand { return &DestroyCommand{BaseCommand{"destroy"}} }

func (c *DestroyCommand) Help() string {
	return `Usage: terraform destroy [options]

Destroy all remote objects managed by this configuration.

Options:
  -auto-approve    Skip interactive approval (REQUIRED for automation).
  -target=resource Limit scope to a specific resource.
  -var=k=v         Set a variable value.
  -var-file=path   Load variable values from a file.
  -parallelism=n   Parallelism for resource operations.

WARNING: This command will destroy your infrastructure!`
}

func (c *DestroyCommand) Validate(payload map[string]any) (map[string]any, []ValidationError) {
	var errs []ValidationError
	clean := map[string]any{}

	autoApprove, ok := getBool(payload, "auto_approve")
	if !ok || !autoApprove {
		errs = append(errs, ValidationError{"auto_approve", fmt.Sprintf("must be true for destroy — %s", c.Help())})
	} else {
		clean["auto_approve"] = true
	}

	if targets, ok := getStringSlice(payload, "target"); ok {
		clean["target"] = targets
	}
	if vars, ok := getStringSlice(payload, "var"); ok {
		clean["var"] = vars
	}
	if varFile, ok := getString(payload, "var_file"); ok && varFile != "" {
		clean["var_file"] = varFile
	}
	return clean, errs
}
