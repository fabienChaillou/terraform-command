package command

import (
	"github.com/fabienChaillou/terraform-commander/internal/terraform"
)

const untaintHelp = `Usage: terraform untaint [options] ADDRESS

Remove the "tainted" marker from a resource so the next apply will not
recreate it.

Options:
  -allow-missing       Succeed even if the resource does not exist.
  -lock=true|false     Acquire a state lock before changing state (default true).
  -lock-timeout=DUR    How long to wait for the lock (e.g. 30s, 5m).

Example:
  terraform untaint aws_instance.web
`

// UntaintCommand implements terraform.Command for "terraform untaint".
//
// Untaint accepts the exact same flag set as taint, so the validation and
// argument-building logic is shared via validateTaintLike / buildTaintLikeArgs.
type UntaintCommand struct{}

func (c *UntaintCommand) Help() string { return untaintHelp }

func (c *UntaintCommand) Validate(req *terraform.Request) []terraform.ValidationError {
	return validateTaintLike(req, "untaint")
}

func (c *UntaintCommand) BuildArgs(req *terraform.Request) []string {
	return buildTaintLikeArgs(req, "untaint")
}
