package command

import (
	"strings"
	"testing"

	"github.com/fabienChaillou/terraform-commander/internal/terraform"
)

// ── Validate ──────────────────────────────────────────────────────────────────

func TestApplyValidate(t *testing.T) {
	cmd := &ApplyCommand{}

	tests := []struct {
		name      string
		req       *terraform.Request
		wantErrs  int
		errFields []string
	}{
		{
			name:      "no args returns error",
			req:       &terraform.Request{Action: "apply"},
			wantErrs:  1,
			errFields: []string{"args"},
		},
		{
			name:     "auto-approve bool is valid",
			req:      &terraform.Request{Action: "apply", Args: map[string]interface{}{"auto-approve": true}},
			wantErrs: 0,
		},
		{
			name:     "var map is valid",
			req:      &terraform.Request{Action: "apply", Args: map[string]interface{}{"var": map[string]interface{}{"env": "staging"}}},
			wantErrs: 0,
		},
		{
			name:     "var-file string is valid",
			req:      &terraform.Request{Action: "apply", Args: map[string]interface{}{"var-file": "staging.tfvars"}},
			wantErrs: 0,
		},
		{
			name:     "target string is valid",
			req:      &terraform.Request{Action: "apply", Args: map[string]interface{}{"target": "module.vpc"}},
			wantErrs: 0,
		},
		{
			name:     "no-color bool is valid",
			req:      &terraform.Request{Action: "apply", Args: map[string]interface{}{"no-color": true}},
			wantErrs: 0,
		},
		{
			name: "all valid flags combined",
			req: &terraform.Request{Action: "apply", Args: map[string]interface{}{
				"auto-approve": true,
				"var":          map[string]interface{}{"env": "prod"},
				"no-color":     true,
			}},
			wantErrs: 0,
		},
		{
			name:      "var as string is invalid",
			req:       &terraform.Request{Action: "apply", Args: map[string]interface{}{"var": "env=prod"}},
			wantErrs:  1,
			errFields: []string{"var"},
		},
		{
			name:      "var-file empty string is invalid",
			req:       &terraform.Request{Action: "apply", Args: map[string]interface{}{"var-file": ""}},
			wantErrs:  1,
			errFields: []string{"var-file"},
		},
		{
			name:      "unknown flag -parallelism is invalid",
			req:       &terraform.Request{Action: "apply", Args: map[string]interface{}{"parallelism": 10}},
			wantErrs:  1,
			errFields: []string{"parallelism"},
		},
		{
			name:      "multiple errors: bad var and unknown flag",
			req:       &terraform.Request{Action: "apply", Args: map[string]interface{}{"var": 42, "unknown": true}},
			wantErrs:  2,
			errFields: []string{"var", "unknown"},
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

func TestApplyBuildArgs(t *testing.T) {
	cmd := &ApplyCommand{}

	tests := []struct {
		name       string
		req        *terraform.Request
		wantArgs   []string
		wantAbsent []string
	}{
		{
			name:     "first arg is apply",
			req:      &terraform.Request{Args: map[string]interface{}{"auto-approve": true}},
			wantArgs: []string{"apply"},
		},
		{
			name:     "auto-approve flag",
			req:      &terraform.Request{Args: map[string]interface{}{"auto-approve": true}},
			wantArgs: []string{"-auto-approve"},
		},
		{
			name:     "var-file flag",
			req:      &terraform.Request{Args: map[string]interface{}{"var-file": "prod.tfvars"}},
			wantArgs: []string{"-var-file=prod.tfvars"},
		},
		{
			name:     "target flag",
			req:      &terraform.Request{Args: map[string]interface{}{"target": "aws_instance.web"}},
			wantArgs: []string{"-target=aws_instance.web"},
		},
		{
			name:     "no-color flag",
			req:      &terraform.Request{Args: map[string]interface{}{"no-color": true}},
			wantArgs: []string{"-no-color"},
		},
		{
			name:       "auto-approve false is omitted",
			req:        &terraform.Request{Args: map[string]interface{}{"auto-approve": false}},
			wantAbsent: []string{"-auto-approve"},
		},
		{
			name:     "var map",
			req:      &terraform.Request{Args: map[string]interface{}{"var": map[string]interface{}{"region": "eu-west-1"}}},
			wantArgs: []string{"-var"},
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

// ── Help ──────────────────────────────────────────────────────────────────────

func TestApplyHelp_MentionsApply(t *testing.T) {
	cmd := &ApplyCommand{}
	if !strings.Contains(cmd.Help(), "apply") {
		t.Error("Help() should mention 'apply'")
	}
}
