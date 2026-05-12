package command

import (
	"strings"
	"testing"

	"github.com/fabienChaillou/terraform-commander/internal/terraform"
)

func TestOutputValidate(t *testing.T) {
	cmd := &OutputCommand{}

	tests := []struct {
		name      string
		req       *terraform.Request
		wantErrs  int
		errFields []string
	}{
		{
			name:     "no args is valid",
			req:      &terraform.Request{Action: "output"},
			wantErrs: 0,
		},
		{
			name:     "json only",
			req:      &terraform.Request{Action: "output", Args: map[string]interface{}{"json": true}},
			wantErrs: 0,
		},
		{
			name:     "raw with name",
			req:      &terraform.Request{Action: "output", Args: map[string]interface{}{"raw": true, "name": "kubeconfig"}},
			wantErrs: 0,
		},
		{
			name:      "raw without name",
			req:       &terraform.Request{Action: "output", Args: map[string]interface{}{"raw": true}},
			wantErrs:  1,
			errFields: []string{"name"},
		},
		{
			name:      "json and raw mutually exclusive",
			req:       &terraform.Request{Action: "output", Args: map[string]interface{}{"json": true, "raw": true, "name": "x"}},
			wantErrs:  1,
			errFields: []string{"json"},
		},
		{
			name:      "name empty string is invalid",
			req:       &terraform.Request{Action: "output", Args: map[string]interface{}{"name": "   "}},
			wantErrs:  1,
			errFields: []string{"name"},
		},
		{
			name:      "unknown flag",
			req:       &terraform.Request{Action: "output", Args: map[string]interface{}{"force": true}},
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

func TestOutputBuildArgs(t *testing.T) {
	cmd := &OutputCommand{}

	t.Run("no args produces just output", func(t *testing.T) {
		args := cmd.BuildArgs(&terraform.Request{})
		if len(args) != 1 || args[0] != "output" {
			t.Errorf("expected [output], got %v", args)
		}
	})
	t.Run("json flag", func(t *testing.T) {
		args := cmd.BuildArgs(&terraform.Request{Args: map[string]interface{}{"json": true}})
		if !containsArg(args, "-json") {
			t.Errorf("expected -json in %v", args)
		}
	})
	t.Run("raw with name", func(t *testing.T) {
		args := cmd.BuildArgs(&terraform.Request{Args: map[string]interface{}{"raw": true, "name": "kube"}})
		if !containsArg(args, "-raw") {
			t.Errorf("expected -raw in %v", args)
		}
		if args[len(args)-1] != "kube" {
			t.Errorf("expected trailing name 'kube' in %v", args)
		}
	})
	t.Run("state flag", func(t *testing.T) {
		args := cmd.BuildArgs(&terraform.Request{Args: map[string]interface{}{"state": "tf.tfstate"}})
		if !containsArg(args, "-state=tf.tfstate") {
			t.Errorf("expected -state=tf.tfstate in %v", args)
		}
	})
}

func TestOutputHelp_MentionsOutput(t *testing.T) {
	if !strings.Contains((&OutputCommand{}).Help(), "output") {
		t.Error("OutputCommand.Help() should mention 'output'")
	}
}
