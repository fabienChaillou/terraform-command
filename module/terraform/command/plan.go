package command

import (
	"fmt"
	"strings"

	"github.com/fabienChaillou/terraform-commander/terraform"
)

const planHelp = `Usage: terraform plan [options]

Show changes required by the current configuration.

Options:
  -var 'NAME=VALUE'      Set an input variable value.
                         Value must be an object: {"var": {"key": "val"}}.
  -var-file=filename     Load variable values from a file.
  -target=resource       Limit the operation to the given resource.
  -out=path              Write a plan file to the given path.
  -destroy               Plan a destroy operation.
  -no-color              Disable color codes in output.

Example:
  terraform plan -var '{"env":"prod"}' -out plan.tfplan
`

var knownPlanFlags = map[string]bool{
	"var":      true,
	"var-file": true,
	"target":   true,
	"out":      true,
	"destroy":  true,
	"no-color": true,
}

// PlanCommand implements terraform.Command for "terraform plan".
type PlanCommand struct{}

func (c *PlanCommand) Help() string { return planHelp }

func (c *PlanCommand) Validate(req *terraform.Request) []terraform.ValidationError {
	var errs []terraform.ValidationError

	if len(req.Args) == 0 {
		return []terraform.ValidationError{{
			Field:   "args",
			Message: "terraform plan requires at least one argument (-var, -var-file, -target, -out, -destroy)",
		}}
	}

	errs = append(errs, unknownFlagErrors(req.Args, knownPlanFlags, "plan")...)

	if v, ok := req.Args["var"]; ok {
		if _, isMap := v.(map[string]interface{}); !isMap {
			errs = append(errs, terraform.ValidationError{
				Field:   "var",
				Message: `-var must be an object: {"var": {"key": "value"}}`,
			})
		}
	}
	if v, ok := req.Args["var-file"]; ok {
		if s, isStr := v.(string); !isStr || strings.TrimSpace(s) == "" {
			errs = append(errs, terraform.ValidationError{Field: "var-file", Message: "-var-file must be a non-empty string"})
		}
	}

	return errs
}

func (c *PlanCommand) BuildArgs(req *terraform.Request) []string {
	args := []string{"plan"}

	if varMap, ok := req.Args["var"].(map[string]interface{}); ok {
		for k, v := range varMap {
			args = append(args, "-var", fmt.Sprintf("%s=%v", k, v))
		}
	}
	if v, ok := req.Args["var-file"].(string); ok {
		args = append(args, "-var-file="+v)
	}
	if v, ok := req.Args["target"].(string); ok {
		args = append(args, "-target="+v)
	}
	if v, ok := req.Args["out"].(string); ok {
		args = append(args, "-out="+v)
	}
	if isTruthy(req.Args["destroy"]) {
		args = append(args, "-destroy")
	}
	if isTruthy(req.Args["no-color"]) {
		args = append(args, "-no-color")
	}

	return args
}
