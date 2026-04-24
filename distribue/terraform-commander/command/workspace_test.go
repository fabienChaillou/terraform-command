package command

import (
	"strings"
	"testing"
)

// ── Validate ──────────────────────────────────────────────────────────────────

func TestWorkspaceValidate_MissingSubcommand(t *testing.T) {
	cmd := &WorkspaceCommand{}
	errs := cmd.Validate(&Request{Action: "workspace"})
	assertErrors(t, errs, 1, []string{"subcommand"})
}

func TestWorkspaceValidate_UnknownSubcommand(t *testing.T) {
	cmd := &WorkspaceCommand{}
	unknowns := []string{"rename", "copy", "push", "BOGUS"}

	for _, sub := range unknowns {
		t.Run(sub, func(t *testing.T) {
			errs := cmd.Validate(&Request{Action: "workspace", Subcommand: sub})
			assertErrors(t, errs, 1, []string{"subcommand"})
		})
	}
}

func TestWorkspaceValidate_New(t *testing.T) {
	cmd := &WorkspaceCommand{}

	tests := []struct {
		name      string
		req       *Request
		wantErrs  int
		errFields []string
	}{
		{
			name:      "missing name",
			req:       &Request{Action: "workspace", Subcommand: "new", Args: map[string]interface{}{}},
			wantErrs:  1,
			errFields: []string{"name"},
		},
		{
			name:      "blank name",
			req:       &Request{Action: "workspace", Subcommand: "new", Args: map[string]interface{}{"name": "   "}},
			wantErrs:  1,
			errFields: []string{"name"},
		},
		{
			name:      "name with spaces",
			req:       &Request{Action: "workspace", Subcommand: "new", Args: map[string]interface{}{"name": "my workspace"}},
			wantErrs:  1,
			errFields: []string{"name"},
		},
		{
			name:      "name with tab",
			req:       &Request{Action: "workspace", Subcommand: "new", Args: map[string]interface{}{"name": "my\tws"}},
			wantErrs:  1,
			errFields: []string{"name"},
		},
		{
			name:     "valid name only",
			req:      &Request{Action: "workspace", Subcommand: "new", Args: map[string]interface{}{"name": "staging"}},
			wantErrs: 0,
		},
		{
			name:     "valid name + state path",
			req:      &Request{Action: "workspace", Subcommand: "new", Args: map[string]interface{}{"name": "staging", "state": "/tmp/tf.tfstate"}},
			wantErrs: 0,
		},
		{
			name:      "state empty string is invalid",
			req:       &Request{Action: "workspace", Subcommand: "new", Args: map[string]interface{}{"name": "staging", "state": ""}},
			wantErrs:  1,
			errFields: []string{"state"},
		},
		{
			name:      "state non-string is invalid",
			req:       &Request{Action: "workspace", Subcommand: "new", Args: map[string]interface{}{"name": "staging", "state": 42}},
			wantErrs:  1,
			errFields: []string{"state"},
		},
		{
			name:      "unknown flag",
			req:       &Request{Action: "workspace", Subcommand: "new", Args: map[string]interface{}{"name": "staging", "force": true}},
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

func TestWorkspaceValidate_Select(t *testing.T) {
	cmd := &WorkspaceCommand{}

	tests := []struct {
		name      string
		req       *Request
		wantErrs  int
		errFields []string
	}{
		{
			name:      "missing name",
			req:       &Request{Action: "workspace", Subcommand: "select", Args: map[string]interface{}{}},
			wantErrs:  1,
			errFields: []string{"name"},
		},
		{
			name:     "valid name",
			req:      &Request{Action: "workspace", Subcommand: "select", Args: map[string]interface{}{"name": "prod"}},
			wantErrs: 0,
		},
		{
			name:      "unknown flag",
			req:       &Request{Action: "workspace", Subcommand: "select", Args: map[string]interface{}{"name": "prod", "force": true}},
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

func TestWorkspaceValidate_Delete(t *testing.T) {
	cmd := &WorkspaceCommand{}

	tests := []struct {
		name      string
		req       *Request
		wantErrs  int
		errFields []string
	}{
		{
			name:      "missing name",
			req:       &Request{Action: "workspace", Subcommand: "delete", Args: map[string]interface{}{}},
			wantErrs:  1,
			errFields: []string{"name"},
		},
		{
			name:     "valid name only",
			req:      &Request{Action: "workspace", Subcommand: "delete", Args: map[string]interface{}{"name": "old-env"}},
			wantErrs: 0,
		},
		{
			name:     "valid name + force",
			req:      &Request{Action: "workspace", Subcommand: "delete", Args: map[string]interface{}{"name": "old-env", "force": true}},
			wantErrs: 0,
		},
		{
			name:      "unknown flag",
			req:       &Request{Action: "workspace", Subcommand: "delete", Args: map[string]interface{}{"name": "old-env", "dry-run": true}},
			wantErrs:  1,
			errFields: []string{"dry-run"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertErrors(t, cmd.Validate(tt.req), tt.wantErrs, tt.errFields)
		})
	}
}

func TestWorkspaceValidate_ListAndShow(t *testing.T) {
	cmd := &WorkspaceCommand{}

	for _, sub := range []string{"list", "show"} {
		t.Run(sub+" no args is valid", func(t *testing.T) {
			errs := cmd.Validate(&Request{Action: "workspace", Subcommand: sub})
			assertErrors(t, errs, 0, nil)
		})

		t.Run(sub+" with extra args is invalid", func(t *testing.T) {
			req := &Request{
				Action:     "workspace",
				Subcommand: sub,
				Args:       map[string]interface{}{"extra": "val"},
			}
			errs := cmd.Validate(req)
			if len(errs) == 0 {
				t.Errorf("workspace %s with args should produce an error", sub)
			}
		})
	}
}

// ── BuildArgs ─────────────────────────────────────────────────────────────────

func TestWorkspaceBuildArgs_New(t *testing.T) {
	cmd := &WorkspaceCommand{}

	t.Run("name only", func(t *testing.T) {
		req := &Request{Subcommand: "new", Args: map[string]interface{}{"name": "dev"}}
		args := cmd.BuildArgs(req)
		assertFirstTwo(t, args, "workspace", "new")
		if !containsArg(args, "dev") {
			t.Errorf("expected workspace name 'dev' in %v", args)
		}
		// Name must be the last positional arg, not a -flag.
		if last := args[len(args)-1]; last != "dev" {
			t.Errorf("workspace name should be last positional arg, got %v", args)
		}
	})

	t.Run("name + state", func(t *testing.T) {
		req := &Request{Subcommand: "new", Args: map[string]interface{}{"name": "dev", "state": "/tmp/tf.tfstate"}}
		args := cmd.BuildArgs(req)
		if !hasArgPrefix(args, "-state=") {
			t.Errorf("expected -state= flag in %v", args)
		}
		if !containsArg(args, "dev") {
			t.Errorf("expected workspace name 'dev' in %v", args)
		}
		// name must come after -state=
		stateIdx, nameIdx := -1, -1
		for i, a := range args {
			if strings.HasPrefix(a, "-state=") {
				stateIdx = i
			}
			if a == "dev" {
				nameIdx = i
			}
		}
		if stateIdx >= nameIdx {
			t.Errorf("-state= (index %d) should precede name (index %d) in %v", stateIdx, nameIdx, args)
		}
	})
}

func TestWorkspaceBuildArgs_Select(t *testing.T) {
	cmd := &WorkspaceCommand{}
	req := &Request{Subcommand: "select", Args: map[string]interface{}{"name": "prod"}}
	args := cmd.BuildArgs(req)
	assertFirstTwo(t, args, "workspace", "select")
	if !containsArg(args, "prod") {
		t.Errorf("expected workspace name 'prod' in %v", args)
	}
}

func TestWorkspaceBuildArgs_Delete(t *testing.T) {
	cmd := &WorkspaceCommand{}

	t.Run("without force", func(t *testing.T) {
		req := &Request{Subcommand: "delete", Args: map[string]interface{}{"name": "old"}}
		args := cmd.BuildArgs(req)
		assertFirstTwo(t, args, "workspace", "delete")
		if containsArg(args, "-force") {
			t.Errorf("unexpected -force in %v", args)
		}
		if !containsArg(args, "old") {
			t.Errorf("expected workspace name 'old' in %v", args)
		}
	})

	t.Run("with force", func(t *testing.T) {
		req := &Request{Subcommand: "delete", Args: map[string]interface{}{"name": "old", "force": true}}
		args := cmd.BuildArgs(req)
		if !containsArg(args, "-force") {
			t.Errorf("expected -force in %v", args)
		}
		// name must come after -force
		forceIdx, nameIdx := -1, -1
		for i, a := range args {
			if a == "-force" {
				forceIdx = i
			}
			if a == "old" {
				nameIdx = i
			}
		}
		if forceIdx >= nameIdx {
			t.Errorf("-force (index %d) should precede name (index %d)", forceIdx, nameIdx)
		}
	})
}

func TestWorkspaceBuildArgs_List(t *testing.T) {
	cmd := &WorkspaceCommand{}
	req := &Request{Subcommand: "list"}
	args := cmd.BuildArgs(req)
	assertFirstTwo(t, args, "workspace", "list")
	if len(args) != 2 {
		t.Errorf("workspace list should produce exactly 2 args, got %v", args)
	}
}

func TestWorkspaceBuildArgs_Show(t *testing.T) {
	cmd := &WorkspaceCommand{}
	req := &Request{Subcommand: "show"}
	args := cmd.BuildArgs(req)
	assertFirstTwo(t, args, "workspace", "show")
	if len(args) != 2 {
		t.Errorf("workspace show should produce exactly 2 args, got %v", args)
	}
}

// ── Help ──────────────────────────────────────────────────────────────────────

func TestWorkspaceHelp_MentionsWorkspace(t *testing.T) {
	cmd := &WorkspaceCommand{}
	if !strings.Contains(cmd.Help(), "workspace") {
		t.Error("Help() should mention 'workspace'")
	}
}

// ── local helpers ─────────────────────────────────────────────────────────────

func assertFirstTwo(t *testing.T, args []string, first, second string) {
	t.Helper()
	if len(args) < 2 {
		t.Fatalf("expected at least 2 args, got %v", args)
	}
	if args[0] != first {
		t.Errorf("args[0] = %q, want %q", args[0], first)
	}
	if args[1] != second {
		t.Errorf("args[1] = %q, want %q", args[1], second)
	}
}

