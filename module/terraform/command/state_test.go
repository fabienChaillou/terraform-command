package command

import (
	"strings"
	"testing"

	"github.com/fabienChaillou/terraform-commander/terraform"
)

// ── Validate ──────────────────────────────────────────────────────────────────

func TestStateValidate_MissingSubcommand(t *testing.T) {
	cmd := &StateCommand{}
	errs := cmd.Validate(&terraform.Request{Action: "state"})
	assertErrors(t, errs, 1, []string{"subcommand"})
}

func TestStateValidate_UnknownSubcommand(t *testing.T) {
	cmd := &StateCommand{}
	for _, sub := range []string{"replace-provider", "BOGUS", "?"} {
		t.Run(sub, func(t *testing.T) {
			errs := cmd.Validate(&terraform.Request{Action: "state", Subcommand: sub})
			assertErrors(t, errs, 1, []string{"subcommand"})
		})
	}
}

func TestStateValidate_List(t *testing.T) {
	cmd := &StateCommand{}

	tests := []struct {
		name      string
		req       *terraform.Request
		wantErrs  int
		errFields []string
	}{
		{
			name:     "no args is valid",
			req:      &terraform.Request{Action: "state", Subcommand: "list"},
			wantErrs: 0,
		},
		{
			name:     "string address is valid",
			req:      &terraform.Request{Action: "state", Subcommand: "list", Args: map[string]interface{}{"address": "aws_vpc.main"}},
			wantErrs: 0,
		},
		{
			name:     "slice address is valid",
			req:      &terraform.Request{Action: "state", Subcommand: "list", Args: map[string]interface{}{"address": []interface{}{"aws_vpc.main", "aws_subnet.a"}}},
			wantErrs: 0,
		},
		{
			name:     "id flag is valid",
			req:      &terraform.Request{Action: "state", Subcommand: "list", Args: map[string]interface{}{"id": "vpc-1234"}},
			wantErrs: 0,
		},
		{
			name:      "id empty string is invalid",
			req:       &terraform.Request{Action: "state", Subcommand: "list", Args: map[string]interface{}{"id": ""}},
			wantErrs:  1,
			errFields: []string{"id"},
		},
		{
			name:      "state empty string is invalid",
			req:       &terraform.Request{Action: "state", Subcommand: "list", Args: map[string]interface{}{"state": ""}},
			wantErrs:  1,
			errFields: []string{"state"},
		},
		{
			name:      "unknown flag",
			req:       &terraform.Request{Action: "state", Subcommand: "list", Args: map[string]interface{}{"force": true}},
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

func TestStateValidate_Show(t *testing.T) {
	cmd := &StateCommand{}

	tests := []struct {
		name      string
		req       *terraform.Request
		wantErrs  int
		errFields []string
	}{
		{
			name:      "missing address",
			req:       &terraform.Request{Action: "state", Subcommand: "show", Args: map[string]interface{}{}},
			wantErrs:  1,
			errFields: []string{"address"},
		},
		{
			name:     "valid address",
			req:      &terraform.Request{Action: "state", Subcommand: "show", Args: map[string]interface{}{"address": "aws_vpc.main"}},
			wantErrs: 0,
		},
		{
			name:      "unknown flag",
			req:       &terraform.Request{Action: "state", Subcommand: "show", Args: map[string]interface{}{"address": "aws_vpc.main", "force": true}},
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

func TestStateValidate_Mv(t *testing.T) {
	cmd := &StateCommand{}

	t.Run("missing source and destination", func(t *testing.T) {
		req := &terraform.Request{Action: "state", Subcommand: "mv", Args: map[string]interface{}{}}
		errs := cmd.Validate(req)
		assertErrors(t, errs, 2, []string{"source", "destination"})
	})
	t.Run("valid src + dst", func(t *testing.T) {
		req := &terraform.Request{Action: "state", Subcommand: "mv", Args: map[string]interface{}{
			"source": "aws_vpc.old", "destination": "aws_vpc.new",
		}}
		assertErrors(t, cmd.Validate(req), 0, nil)
	})
	t.Run("unknown flag", func(t *testing.T) {
		req := &terraform.Request{Action: "state", Subcommand: "mv", Args: map[string]interface{}{
			"source": "a", "destination": "b", "force": true,
		}}
		assertErrors(t, cmd.Validate(req), 1, []string{"force"})
	})
}

func TestStateValidate_Rm(t *testing.T) {
	cmd := &StateCommand{}

	t.Run("missing address", func(t *testing.T) {
		req := &terraform.Request{Action: "state", Subcommand: "rm", Args: map[string]interface{}{}}
		assertErrors(t, cmd.Validate(req), 1, []string{"address"})
	})
	t.Run("valid single address", func(t *testing.T) {
		req := &terraform.Request{Action: "state", Subcommand: "rm", Args: map[string]interface{}{"address": "aws_vpc.main"}}
		assertErrors(t, cmd.Validate(req), 0, nil)
	})
	t.Run("valid multi address", func(t *testing.T) {
		req := &terraform.Request{Action: "state", Subcommand: "rm", Args: map[string]interface{}{
			"address": []interface{}{"a", "b"},
		}}
		assertErrors(t, cmd.Validate(req), 0, nil)
	})
}

func TestStateValidate_PullAndPush(t *testing.T) {
	cmd := &StateCommand{}

	t.Run("pull no args is valid", func(t *testing.T) {
		assertErrors(t, cmd.Validate(&terraform.Request{Action: "state", Subcommand: "pull"}), 0, nil)
	})
	t.Run("pull with extra args is invalid", func(t *testing.T) {
		errs := cmd.Validate(&terraform.Request{Action: "state", Subcommand: "pull", Args: map[string]interface{}{"force": true}})
		if len(errs) == 0 {
			t.Error("state pull with args should be invalid")
		}
	})
	t.Run("push missing path", func(t *testing.T) {
		assertErrors(t, cmd.Validate(&terraform.Request{Action: "state", Subcommand: "push", Args: map[string]interface{}{}}), 1, []string{"path"})
	})
	t.Run("push valid path", func(t *testing.T) {
		assertErrors(t, cmd.Validate(&terraform.Request{Action: "state", Subcommand: "push", Args: map[string]interface{}{"path": "tf.tfstate"}}), 0, nil)
	})
	t.Run("push with force", func(t *testing.T) {
		assertErrors(t, cmd.Validate(&terraform.Request{Action: "state", Subcommand: "push", Args: map[string]interface{}{"path": "tf.tfstate", "force": true}}), 0, nil)
	})
}

// ── BuildArgs ─────────────────────────────────────────────────────────────────

func TestStateBuildArgs_List(t *testing.T) {
	cmd := &StateCommand{}

	t.Run("no args", func(t *testing.T) {
		args := cmd.BuildArgs(&terraform.Request{Action: "state", Subcommand: "list"})
		assertFirstTwo(t, args, "state", "list")
		if len(args) != 2 {
			t.Errorf("list with no args should be exactly 2 args, got %v", args)
		}
	})
	t.Run("with address slice", func(t *testing.T) {
		args := cmd.BuildArgs(&terraform.Request{Action: "state", Subcommand: "list", Args: map[string]interface{}{
			"address": []interface{}{"a", "b"},
		}})
		if !containsArg(args, "a") || !containsArg(args, "b") {
			t.Errorf("expected addresses 'a' and 'b' in %v", args)
		}
	})
	t.Run("state and id flags", func(t *testing.T) {
		args := cmd.BuildArgs(&terraform.Request{Action: "state", Subcommand: "list", Args: map[string]interface{}{
			"state": "tf.tfstate", "id": "vpc-1",
		}})
		if !containsArg(args, "-state=tf.tfstate") || !containsArg(args, "-id=vpc-1") {
			t.Errorf("expected -state=tf.tfstate and -id=vpc-1 in %v", args)
		}
	})
}

func TestStateBuildArgs_Show(t *testing.T) {
	cmd := &StateCommand{}
	req := &terraform.Request{Subcommand: "show", Args: map[string]interface{}{"address": "aws_vpc.main"}}
	args := cmd.BuildArgs(req)
	assertFirstTwo(t, args, "state", "show")
	if args[len(args)-1] != "aws_vpc.main" {
		t.Errorf("expected last arg to be the address, got %v", args)
	}
}

func TestStateBuildArgs_Mv(t *testing.T) {
	cmd := &StateCommand{}
	req := &terraform.Request{Subcommand: "mv", Args: map[string]interface{}{
		"source": "aws_vpc.old", "destination": "aws_vpc.new",
		"state": "tf.tfstate", "state-out": "out.tfstate",
		"dry-run": true,
	}}
	args := cmd.BuildArgs(req)
	assertFirstTwo(t, args, "state", "mv")
	if !containsArg(args, "-dry-run") || !hasArgPrefix(args, "-state=") || !hasArgPrefix(args, "-state-out=") {
		t.Errorf("expected -dry-run/-state=/-state-out= in %v", args)
	}
	// Positional ordering: source then destination, last two elements.
	if args[len(args)-2] != "aws_vpc.old" || args[len(args)-1] != "aws_vpc.new" {
		t.Errorf("expected source then destination at end, got %v", args)
	}
}

func TestStateBuildArgs_Rm(t *testing.T) {
	cmd := &StateCommand{}
	req := &terraform.Request{Subcommand: "rm", Args: map[string]interface{}{
		"address": []interface{}{"a", "b"},
		"dry-run": true,
	}}
	args := cmd.BuildArgs(req)
	assertFirstTwo(t, args, "state", "rm")
	if !containsArg(args, "-dry-run") || !containsArg(args, "a") || !containsArg(args, "b") {
		t.Errorf("expected -dry-run and addresses in %v", args)
	}
}

func TestStateBuildArgs_Pull(t *testing.T) {
	cmd := &StateCommand{}
	args := cmd.BuildArgs(&terraform.Request{Subcommand: "pull"})
	assertFirstTwo(t, args, "state", "pull")
	if len(args) != 2 {
		t.Errorf("state pull should produce exactly 2 args, got %v", args)
	}
}

func TestStateBuildArgs_Push(t *testing.T) {
	cmd := &StateCommand{}
	req := &terraform.Request{Subcommand: "push", Args: map[string]interface{}{
		"path": "tf.tfstate", "force": true,
	}}
	args := cmd.BuildArgs(req)
	assertFirstTwo(t, args, "state", "push")
	if !containsArg(args, "-force") || args[len(args)-1] != "tf.tfstate" {
		t.Errorf("expected -force flag and trailing path, got %v", args)
	}
}

// ── Help ──────────────────────────────────────────────────────────────────────

func TestStateHelp_MentionsState(t *testing.T) {
	cmd := &StateCommand{}
	if !strings.Contains(cmd.Help(), "state") {
		t.Error("Help() should mention 'state'")
	}
}
