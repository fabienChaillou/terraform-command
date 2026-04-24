package command

import (
	"fmt"
	"strings"
)

const applyHelp = `Usage: terraform apply [options]

Create or update infrastructure.

Options:
  -var 'NAME=VALUE'      Set an input variable value.
  -var-file=filename     Load variable values from a file.
  -target=resource       Limit the operation to the given resource.
  -auto-approve          Skip interactive approval.
  -no-color              Disable color codes in output.

Example:
  terraform apply -var '{"env":"prod"}' -auto-approve
`

var knownApplyFlags = map[string]bool{
	"var":          true,
	"var-file":     true,
	"target":       true,
	"auto-approve": true,
	"no-color":     true,
}

// ApplyCommand implements Command for "terraform apply".
type ApplyCommand struct{}

func (c *ApplyCommand) Help() string { return applyHelp }

func (c *ApplyCommand) Validate(req *Request) []ValidationError {
	var errs []ValidationError

	if len(req.Args) == 0 {
		return []ValidationError{{
			Field:   "args",
			Message: "terraform apply requires at least one argument (-var, -var-file, -auto-approve)",
		}}
	}

	errs = append(errs, unknownFlagErrors(req.Args, knownApplyFlags, "apply")...)

	if v, ok := req.Args["var"]; ok {
		if _, isMap := v.(map[string]interface{}); !isMap {
			errs = append(errs, ValidationError{
				Field:   "var",
				Message: `-var must be an object: {"var": {"key": "value"}}`,
			})
		}
	}
	if v, ok := req.Args["var-file"]; ok {
		if s, isStr := v.(string); !isStr || strings.TrimSpace(s) == "" {
			errs = append(errs, ValidationError{Field: "var-file", Message: "-var-file must be a non-empty string"})
		}
	}

	return errs
}

func (c *ApplyCommand) BuildArgs(req *Request) []string {
	args := []string{"apply"}

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
