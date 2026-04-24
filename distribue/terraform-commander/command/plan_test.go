package command

import (
	"strings"
	"testing"
)

// ── Validate ──────────────────────────────────────────────────────────────────

func TestPlanValidate(t *testing.T) {
	cmd := &PlanCommand{}

	tests := []struct {
		name      string
		req       *Request
		wantErrs  int
		errFields []string
	}{
		{
			name:      "no args returns error",
			req:       &Request{Action: "plan"},
			wantErrs:  1,
			errFields: []string{"args"},
		},
		{
			name:     "var map is valid",
			req:      &Request{Action: "plan", Args: map[string]interface{}{"var": map[string]interface{}{"env": "prod"}}},
			wantErrs: 0,
		},
		{
			name:     "var-file string is valid",
			req:      &Request{Action: "plan", Args: map[string]interface{}{"var-file": "vars.tfvars"}},
			wantErrs: 0,
		},
		{
			name:     "target string is valid",
			req:      &Request{Action: "plan", Args: map[string]interface{}{"target": "aws_vpc.main"}},
			wantErrs: 0,
		},
		{
			name:     "out string is valid",
			req:      &Request{Action: "plan", Args: map[string]interface{}{"out": "plan.tfplan"}},
			wantErrs: 0,
		},
		{
			name:     "destroy bool is valid",
			req:      &Request{Action: "plan", Args: map[string]interface{}{"destroy": true}},
			wantErrs: 0,
		},
		{
			name:     "no-color bool is valid",
			req:      &Request{Action: "plan", Args: map[string]interface{}{"no-color": true}},
			wantErrs: 0,
		},
		{
			name:      "var as string is invalid (must be map)",
			req:       &Request{Action: "plan", Args: map[string]interface{}{"var": "env=prod"}},
			wantErrs:  1,
			errFields: []string{"var"},
		},
		{
			name:      "var as bool is invalid",
			req:       &Request{Action: "plan", Args: map[string]interface{}{"var": true}},
			wantErrs:  1,
			errFields: []string{"var"},
		},
		{
			name:      "var-file empty string is invalid",
			req:       &Request{Action: "plan", Args: map[string]interface{}{"var-file": ""}},
			wantErrs:  1,
			errFields: []string{"var-file"},
		},
		{
			name:      "var-file whitespace-only is invalid",
			req:       &Request{Action: "plan", Args: map[string]interface{}{"var-file": "   "}},
			wantErrs:  1,
			errFields: []string{"var-file"},
		},
		{
			name:      "var-file non-string is invalid",
			req:       &Request{Action: "plan", Args: map[string]interface{}{"var-file": 42}},
			wantErrs:  1,
			errFields: []string{"var-file"},
		},
		{
			name:      "unknown flag produces error",
			req:       &Request{Action: "plan", Args: map[string]interface{}{"refresh": true}},
			wantErrs:  1,
			errFields: []string{"refresh"},
		},
		{
			name:      "multiple validation errors",
			req:       &Request{Action: "plan", Args: map[string]interface{}{"var": "bad", "bad-flag": true}},
			wantErrs:  2,
			errFields: []string{"var", "bad-flag"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := cmd.Validate(tt.req)
			assertErrors(t, errs, tt.wantErrs, tt.errFields)
		})
	}
}

// ── BuildArgs ─────────────────────────────────────────────────────────────────

func TestPlanBuildArgs(t *testing.T) {
	cmd := &PlanCommand{}

	tests := []struct {
		name       string
		req        *Request
		wantArgs   []string
		wantAbsent []string
	}{
		{
			name:     "first arg is plan",
			req:      &Request{Args: map[string]interface{}{"no-color": true}},
			wantArgs: []string{"plan"},
		},
		{
			name: "var map expands to -var key=value pairs",
			req:  &Request{Args: map[string]interface{}{"var": map[string]interface{}{"env": "prod"}}},
			// -var must be followed immediately by the key=value pair
			wantArgs: []string{"-var"},
		},
		{
			name:     "var-file flag",
			req:      &Request{Args: map[string]interface{}{"var-file": "vars.tfvars"}},
			wantArgs: []string{"-var-file=vars.tfvars"},
		},
		{
			name:     "target flag",
			req:      &Request{Args: map[string]interface{}{"target": "aws_vpc.main"}},
			wantArgs: []string{"-target=aws_vpc.main"},
		},
		{
			name:     "out flag",
			req:      &Request{Args: map[string]interface{}{"out": "plan.out"}},
			wantArgs: []string{"-out=plan.out"},
		},
		{
			name:     "destroy flag",
			req:      &Request{Args: map[string]interface{}{"destroy": true}},
			wantArgs: []string{"-destroy"},
		},
		{
			name:     "no-color flag",
			req:      &Request{Args: map[string]interface{}{"no-color": true}},
			wantArgs: []string{"-no-color"},
		},
		{
			name:        "false flags are omitted",
			req:         &Request{Args: map[string]interface{}{"destroy": false, "no-color": false}},
			wantAbsent:  []string{"-destroy", "-no-color"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := cmd.BuildArgs(tt.req)
			for _, want := range tt.wantArgs {
				if !containsArg(args, want) {
					t.Errorf("missing arg %q in %v", want, args)
				}
			}
			for _, absent := range tt.wantAbsent {
				if containsArg(args, absent) {
					t.Errorf("unexpected arg %q in %v", absent, args)
				}
			}
		})
	}
}

// TestPlanBuildArgs_VarKeyValue checks that -var env=prod pair is correctly formed.
func TestPlanBuildArgs_VarKeyValue(t *testing.T) {
	cmd := &PlanCommand{}
	req := &Request{Args: map[string]interface{}{
		"var": map[string]interface{}{"env": "prod"},
	}}
	args := cmd.BuildArgs(req)

	// Find index of "-var" then check the next element.
	for i, a := range args {
		if a == "-var" {
			if i+1 >= len(args) {
				t.Fatal("-var flag has no following value")
			}
			if !strings.Contains(args[i+1], "=") {
				t.Errorf("-var value %q should contain '='", args[i+1])
			}
			return
		}
	}
	t.Error("-var flag not found in BuildArgs output")
}

// ── Help ──────────────────────────────────────────────────────────────────────

func TestPlanHelp_MentionsPlan(t *testing.T) {
	cmd := &PlanCommand{}
	if !strings.Contains(cmd.Help(), "plan") {
		t.Error("Help() should mention 'plan'")
	}
}

// assertErrors is a shared helper used by plan, apply, destroy tests.
func assertErrors(t *testing.T, errs []ValidationError, wantCount int, wantFields []string) {
	t.Helper()
	if wantCount == 0 {
		if len(errs) != 0 {
			t.Fatalf("expected no validation errors, got %d: %+v", len(errs), errs)
		}
		return
	}
	if len(errs) == 0 {
		t.Fatal("expected validation errors, got none")
	}
	if len(errs) != wantCount {
		t.Errorf("expected %d validation error(s), got %d: %+v", wantCount, len(errs), errs)
	}
	for _, field := range wantFields {
		if !containsField(errs, field) {
			t.Errorf("expected error on field %q, errors = %+v", field, errs)
		}
	}
}
