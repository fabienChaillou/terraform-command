package command

import (
	"fmt"
	"strings"

	"github.com/fabienChaillou/terraform-commander/internal/terraform"
)

const destroyHelp = `Usage: terraform destroy [options]

Destroy previously-created infrastructure.

Options:
  -var 'NAME=VALUE'      Set an input variable value.
  -var-file=filename     Load variable values from a file.
  -target=resource       Limit the operation to the given resource.
  -auto-approve          Skip interactive approval.
  -no-color              Disable color codes in output.

Example:
  terraform destroy -var '{"env":"staging"}' -auto-approve
`

var knownDestroyFlags = map[string]bool{
	"var":          true,
	"var-file":     true,
	"target":       true,
	"auto-approve": true,
	"no-color":     true,
}

// DestroyCommand implements terraform.Command for "terraform destroy".
type DestroyCommand struct{}

func (c *DestroyCommand) Help() string { return destroyHelp }

func (c *DestroyCommand) Validate(req *terraform.Request) []terraform.ValidationError {
	var errs []terraform.ValidationError

	if len(req.Args) == 0 {
		return []terraform.ValidationError{{
			Field:   "args",
			Message: "terraform destroy requires at least one argument (-var, -var-file, -auto-approve)",
		}}
	}

	errs = append(errs, unknownFlagErrors(req.Args, knownDestroyFlags, "destroy")...)

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

func (c *DestroyCommand) BuildArgs(req *terraform.Request) []string {
	args := []string{"destroy"}

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
	if isTruthy(req.Args["auto-approve"]) {
		args = append(args, "-auto-approve")
	}
	if isTruthy(req.Args["no-color"]) {
		args = append(args, "-no-color")
	}

	return args
}
