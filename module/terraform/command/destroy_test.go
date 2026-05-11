package command

import (
	"strings"
	"testing"

	"github.com/fabienChaillou/terraform-commander/internal/terraform"
)

// ── Validate ──────────────────────────────────────────────────────────────────

func TestDestroyValidate(t *testing.T) {
	cmd := &DestroyCommand{}

	tests := []struct {
		name      string
		req       *terraform.Request
		wantErrs  int
		errFields []string
	}{
		{
			name:      "no args returns error",
			req:       &terraform.Request{Action: "destroy"},
			wantErrs:  1,
			errFields: []string{"args"},
		},
		{
			name:     "auto-approve bool is valid",
			req:      &terraform.Request{Action: "destroy", Args: map[string]interface{}{"auto-approve": true}},
			wantErrs: 0,
		},
		{
			name:     "var map is valid",
			req:      &terraform.Request{Action: "destroy", Args: map[string]interface{}{"var": map[string]interface{}{"env": "dev"}}},
			wantErrs: 0,
		},
		{
			name:     "var-file string is valid",
			req:      &terraform.Request{Action: "destroy", Args: map[string]interface{}{"var-file": "dev.tfvars"}},
			wantErrs: 0,
		},
		{
			name:     "target string is valid",
			req:      &terraform.Request{Action: "destroy", Args: map[string]interface{}{"target": "aws_s3_bucket.logs"}},
			wantErrs: 0,
		},
		{
			name:     "no-color bool is valid",
			req:      &terraform.Request{Action: "destroy", Args: map[string]interface{}{"no-color": true}},
			wantErrs: 0,
		},
		{
			name: "all valid flags combined",
			req: &terraform.Request{Action: "destroy", Args: map[string]interface{}{
				"auto-approve": true,
				"var":          map[string]interface{}{"env": "dev"},
				"no-color":     true,
			}},
			wantErrs: 0,
		},
		{
			name:      "var as non-map is invalid",
			req:       &terraform.Request{Action: "destroy", Args: map[string]interface{}{"var": []string{"env=dev"}}},
			wantErrs:  1,
			errFields: []string{"var"},
		},
		{
			name:      "var-file empty string is invalid",
			req:       &terraform.Request{Action: "destroy", Args: map[string]interface{}{"var-file": ""}},
			wantErrs:  1,
			errFields: []string{"var-file"},
		},
		{
			name:      "unknown flag produces error",
			req:       &terraform.Request{Action: "destroy", Args: map[string]interface{}{"state": "terraform.tfstate"}},
			wantErrs:  1,
			errFields: []string{"state"},
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

func TestDestroyBuildArgs(t *testing.T) {
	cmd := &DestroyCommand{}

	tests := []struct {
		name       string
		req        *terraform.Request
		wantArgs   []string
		wantAbsent []string
	}{
		{
			name:     "first arg is destroy",
			req:      &terraform.Request{Args: map[string]interface{}{"auto-approve": true}},
			wantArgs: []string{"destroy"},
		},
		{
			name:     "auto-approve flag",
			req:      &terraform.Request{Args: map[string]interface{}{"auto-approve": true}},
			wantArgs: []string{"destroy", "-auto-approve"},
		},
		{
			name:     "var-file flag",
			req:      &terraform.Request{Args: map[string]interface{}{"var-file": "dev.tfvars"}},
			wantArgs: []string{"-var-file=dev.tfvars"},
		},
		{
			name:     "target flag",
			req:      &terraform.Request{Args: map[string]interface{}{"target": "aws_vpc.main"}},
			wantArgs: []string{"-target=aws_vpc.main"},
		},
		{
			name:     "no-color flag",
			req:      &terraform.Request{Args: map[string]interface{}{"no-color": true}},
			wantArgs: []string{"-no-color"},
		},
		{
			name:       "false flags are omitted",
			req:        &terraform.Request{Args: map[string]interface{}{"auto-approve": false, "no-color": false}},
			wantAbsent: []string{"-auto-approve", "-no-color"},
		},
		{
			name:     "var map expands",
			req:      &terraform.Request{Args: map[string]interface{}{"var": map[string]interface{}{"env": "dev"}}},
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

func TestDestroyHelp_MentionsDestroy(t *testing.T) {
	cmd := &DestroyCommand{}
	if !strings.Contains(cmd.Help(), "destroy") {
		t.Error("Help() should mention 'destroy'")
	}
}
