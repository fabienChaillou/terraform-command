package command

import (
	"strings"

	"github.com/fabienChaillou/terraform-commander/terraform"
)

const outputHelp = `Usage: terraform output [options] [NAME]

Read an output value from a Terraform state file.

Options:
  -json              Print outputs as JSON.
  -raw               Print the value of a single named output (string only).
  -state=PATH        Path to a state file.
  -no-color          Disable color codes in output.

Arguments:
  NAME   Optional output name.  Without it, all outputs are printed.

Example:
  terraform output -json
  terraform output -raw kubeconfig
`

var knownOutputFlags = map[string]bool{
	"name":     true,
	"json":     true,
	"raw":      true,
	"state":    true,
	"no-color": true,
}

// OutputCommand implements terraform.Command for "terraform output".
type OutputCommand struct{}

func (c *OutputCommand) Help() string { return outputHelp }

func (c *OutputCommand) Validate(req *terraform.Request) []terraform.ValidationError {
	var errs []terraform.ValidationError

	// -json and -raw are mutually exclusive in the Terraform CLI.
	if isTruthy(req.Args["json"]) && isTruthy(req.Args["raw"]) {
		errs = append(errs, terraform.ValidationError{
			Field:   "json",
			Message: "-json and -raw are mutually exclusive",
		})
	}
	// -raw requires NAME (a single string output) to be useful.
	if isTruthy(req.Args["raw"]) {
		if e := requireStringArg(req.Args, "name",
			`-raw requires an output name: {"name": "<output>"}`); e != nil {
			errs = append(errs, *e)
		}
	}
	if v, ok := req.Args["name"]; ok {
		if s, isStr := v.(string); !isStr || strings.TrimSpace(s) == "" {
			errs = append(errs, terraform.ValidationError{Field: "name", Message: "name must be a non-empty string"})
		}
	}
	if v, ok := req.Args["state"]; ok {
		if s, isStr := v.(string); !isStr || strings.TrimSpace(s) == "" {
			errs = append(errs, terraform.ValidationError{Field: "state", Message: "-state must be a non-empty file path"})
		}
	}

	errs = append(errs, unknownFlagErrors(req.Args, knownOutputFlags, "output")...)
	return errs
}

func (c *OutputCommand) BuildArgs(req *terraform.Request) []string {
	args := []string{"output"}

	if isTruthy(req.Args["json"]) {
		args = append(args, "-json")
	}
	if isTruthy(req.Args["raw"]) {
		args = append(args, "-raw")
	}
	if v, ok := req.Args["state"].(string); ok && v != "" {
		args = append(args, "-state="+v)
	}
	if isTruthy(req.Args["no-color"]) {
		args = append(args, "-no-color")
	}
	if name, ok := req.Args["name"].(string); ok && name != "" {
		args = append(args, name)
	}
	return args
}
