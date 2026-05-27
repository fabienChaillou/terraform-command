package command

import "github.com/fabienChaillou/terraform-commander/terraform"

// The Terraform CLI does not expose a top-level "unlock" command — the actual
// verb is "force-unlock".  This package keeps the action name short ("unlock")
// for HTTP callers while emitting the correct CLI verb in BuildArgs.

const unlockHelp = `Usage: terraform force-unlock [-force] LOCK_ID

Forcibly release a state lock held by another operation.  Use only after
confirming that no terraform process is still running against this state.

Options:
  -force   Do not prompt for confirmation before releasing the lock.

Example:
  terraform force-unlock 1234abcd-5678-...
`

var knownUnlockFlags = map[string]bool{
	"lock-id": true,
	"force":   true,
}

// UnlockCommand implements terraform.Command for the "unlock" action.
// The CLI verb emitted by BuildArgs is "force-unlock" — see package doc.
type UnlockCommand struct{}

func (c *UnlockCommand) Help() string { return unlockHelp }

func (c *UnlockCommand) Validate(req *terraform.Request) []terraform.ValidationError {
	var errs []terraform.ValidationError

	if e := requireStringArg(req.Args, "lock-id",
		`unlock requires a lock id: {"lock-id": "<id>"}`); e != nil {
		errs = append(errs, *e)
	}

	errs = append(errs, unknownFlagErrors(req.Args, knownUnlockFlags, "force-unlock")...)
	return errs
}

// BuildArgs emits the CLI sequence:
//
//	["force-unlock", "-force"?, LOCK_ID]
//
// "force-unlock" is the actual verb accepted by the Terraform binary even
// though the action exposed via this package is named "unlock".
func (c *UnlockCommand) BuildArgs(req *terraform.Request) []string {
	args := []string{"force-unlock"}

	if isTruthy(req.Args["force"]) {
		args = append(args, "-force")
	}
	if id, ok := req.Args["lock-id"].(string); ok && id != "" {
		args = append(args, id)
	}
	return args
}
