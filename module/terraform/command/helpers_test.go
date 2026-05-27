package command

import (
	"strings"

	"github.com/fabienChaillou/terraform-commander/terraform"
)

// helpers_test.go centralises the assertion helpers reused by every per-command
// test file (init_test.go, plan_test.go, apply_test.go, destroy_test.go,
// workspace_test.go).
//
// They previously lived inline in each test file under the old monolithic
// command package; moving them here keeps each command test file focused on
// its own subject.

func containsField(errs []terraform.ValidationError, field string) bool {
	for _, e := range errs {
		if e.Field == field {
			return true
		}
	}
	return false
}

func containsArg(args []string, arg string) bool {
	for _, a := range args {
		if a == arg {
			return true
		}
	}
	return false
}

func hasArgPrefix(args []string, prefix string) bool {
	for _, a := range args {
		if strings.HasPrefix(a, prefix) {
			return true
		}
	}
	return false
}

func safeFirst(args []string) string {
	if len(args) == 0 {
		return "<empty>"
	}
	return args[0]
}
