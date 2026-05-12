package command

import (
	"fmt"
	"strings"

	"github.com/fabienChaillou/terraform-commander/internal/terraform"
)

const refreshHelp = `Usage: terraform refresh [options]

Update local state to match real-world infrastructure.

Note: in modern Terraform this is equivalent to "terraform apply -refresh-only";
the dedicated "refresh" verb is kept for backwards compatibility.

Options:
  -var 'NAME=VALUE'      Set an input variable (object: {"var": {"k":"v"}}).
  -var-file=filename     Load variable values from a file.
  -target=resource       Limit the refresh to the given resource.
  -lock-timeout=DUR      How long to wait for the state lock.
  -no-color              Disable color codes in output.

Example:
  terraform refresh -var '{"env":"prod"}'
`

var knownRefreshFlags = map[string]bool{
	"var":          true,
	"var-file":     true,
	"target":       true,
	"lock-timeout": true,
	"no-color":     true,
}

// RefreshCommand implements terraform.Command for "terraform refresh".
//
// Unlike init/plan/apply/destroy, refresh accepts an empty args map — running
// it with no flags is a perfectly valid invocation.  We therefore skip the
// "must have at least one argument" check those commands enforce.
type RefreshCommand struct{}

func (c *RefreshCommand) Help() string { return refreshHelp }

func (c *RefreshCommand) Validate(req *terraform.Request) []terraform.ValidationError {
	var errs []terraform.ValidationError

	errs = append(errs, unknownFlagErrors(req.Args, knownRefreshFlags, "refresh")...)

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
	if v, ok := req.Args["lock-timeout"]; ok {
		if s, isStr := v.(string); !isStr || strings.TrimSpace(s) == "" {
			errs = append(errs, terraform.ValidationError{
				Field:   "lock-timeout",
				Message: "-lock-timeout must be a non-empty duration string (e.g. \"30s\", \"5m\")",
			})
		}
	}
	return errs
}

func (c *RefreshCommand) BuildArgs(req *terraform.Request) []string {
	args := []string{"refresh"}

	if varMap, ok := req.Args["var"].(map[string]interface{}); ok {
		for k, v := range varMap {
			args = append(args, "-var", fmt.Sprintf("%s=%v", k, v))
		}
	}
	if v, ok := req.Args["var-file"].(string); ok && v != "" {
		args = append(args, "-var-file="+v)
	}
	if v, ok := req.Args["target"].(string); ok && v != "" {
		args = append(args, "-target="+v)
	}
	if v, ok := req.Args["lock-timeout"].(string); ok && v != "" {
		args = append(args, "-lock-timeout="+v)
	}
	if isTruthy(req.Args["no-color"]) {
		args = append(args, "-no-color")
	}
	return args
}
