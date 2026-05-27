package command

import "github.com/fabienChaillou/terraform-commander/terraform"

const initHelp = `Usage: terraform init [options]

Initialize a new or existing Terraform working directory.

Options:
  -backend-config=path   Path to a partial backend configuration file.
  -reconfigure           Reconfigure the backend, ignoring saved config.
  -upgrade               Upgrade providers and modules to latest allowed.
  -no-color              Disable color codes in output.

Example:
  terraform init -backend-config=backend.hcl -reconfigure
`

var knownInitFlags = map[string]bool{
	"backend-config": true,
	"reconfigure":    true,
	"upgrade":        true,
	"no-color":       true,
}

// InitCommand implements terraform.Command for "terraform init".
type InitCommand struct{}

func (c *InitCommand) Help() string { return initHelp }

func (c *InitCommand) Validate(req *terraform.Request) []terraform.ValidationError {
	var errs []terraform.ValidationError

	if len(req.Args) == 0 {
		return []terraform.ValidationError{{
			Field:   "args",
			Message: "terraform init requires at least one argument (-backend-config, -reconfigure, -upgrade)",
		}}
	}

	errs = append(errs, unknownFlagErrors(req.Args, knownInitFlags, "init")...)
	return errs
}

func (c *InitCommand) BuildArgs(req *terraform.Request) []string {
	args := []string{"init"}

	if v, ok := req.Args["backend-config"].(string); ok && v != "" {
		args = append(args, "-backend-config="+v)
	}
	if isTruthy(req.Args["reconfigure"]) {
		args = append(args, "-reconfigure")
	}
	if isTruthy(req.Args["upgrade"]) {
		args = append(args, "-upgrade")
	}
	if isTruthy(req.Args["no-color"]) {
		args = append(args, "-no-color")
	}

	return args
}
