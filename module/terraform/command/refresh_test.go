package command

import (
	"strings"
	"testing"

	"github.com/fabienChaillou/terraform-commander/terraform"
)

func TestRefreshValidate(t *testing.T) {
	cmd := &RefreshCommand{}

	tests := []struct {
		name      string
		req       *terraform.Request
		wantErrs  int
		errFields []string
	}{
		{
			name:     "no args is valid",
			req:      &terraform.Request{Action: "refresh"},
			wantErrs: 0,
		},
		{
			name:     "var map is valid",
			req:      &terraform.Request{Action: "refresh", Args: map[string]interface{}{"var": map[string]interface{}{"env": "prod"}}},
			wantErrs: 0,
		},
		{
			name:      "var as string is invalid",
			req:       &terraform.Request{Action: "refresh", Args: map[string]interface{}{"var": "env=prod"}},
			wantErrs:  1,
			errFields: []string{"var"},
		},
		{
			name:      "var-file blank invalid",
			req:       &terraform.Request{Action: "refresh", Args: map[string]interface{}{"var-file": "  "}},
			wantErrs:  1,
			errFields: []string{"var-file"},
		},
		{
			name:     "target only",
			req:      &terraform.Request{Action: "refresh", Args: map[string]interface{}{"target": "aws_vpc.main"}},
			wantErrs: 0,
		},
		{
			name:     "lock-timeout valid",
			req:      &terraform.Request{Action: "refresh", Args: map[string]interface{}{"lock-timeout": "30s"}},
			wantErrs: 0,
		},
		{
			name:      "lock-timeout non-string",
			req:       &terraform.Request{Action: "refresh", Args: map[string]interface{}{"lock-timeout": 30}},
			wantErrs:  1,
			errFields: []string{"lock-timeout"},
		},
		{
			name:      "unknown flag",
			req:       &terraform.Request{Action: "refresh", Args: map[string]interface{}{"force": true}},
			wantErrs:  1,
			errFields: []string{"force"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertErrors(t, cmd.Validate(tt.req), tt.wantErrs, tt.errFields)
		})
	}
}

func TestRefreshBuildArgs(t *testing.T) {
	cmd := &RefreshCommand{}

	t.Run("no args", func(t *testing.T) {
		args := cmd.BuildArgs(&terraform.Request{})
		if len(args) != 1 || args[0] != "refresh" {
			t.Errorf("refresh with no args should produce [refresh], got %v", args)
		}
	})
	t.Run("all flags", func(t *testing.T) {
		args := cmd.BuildArgs(&terraform.Request{Args: map[string]interface{}{
			"var":          map[string]interface{}{"env": "prod"},
			"var-file":     "vars.tfvars",
			"target":       "aws_vpc.main",
			"lock-timeout": "30s",
			"no-color":     true,
		}})
		for _, want := range []string{
			"-var", "-var-file=vars.tfvars", "-target=aws_vpc.main",
			"-lock-timeout=30s", "-no-color",
		} {
			if !containsArg(args, want) {
				t.Errorf("missing arg %q in %v", want, args)
			}
		}
	})
}

func TestRefreshHelp_MentionsRefresh(t *testing.T) {
	if !strings.Contains((&RefreshCommand{}).Help(), "refresh") {
		t.Error("RefreshCommand.Help() should mention 'refresh'")
	}
}
