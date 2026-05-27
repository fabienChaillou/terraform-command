package command

import (
	"strings"
	"testing"

	"github.com/fabienChaillou/terraform-commander/terraform"
)

// ── Validate ──────────────────────────────────────────────────────────────────

func TestInitValidate(t *testing.T) {
	cmd := &InitCommand{}

	tests := []struct {
		name      string
		req       *terraform.Request
		wantErrs  int      // expected number of ValidationErrors (0 = valid)
		errFields []string // at least these fields must appear in errors
	}{
		{
			name:      "no args returns error",
			req:       &terraform.Request{Action: "init"},
			wantErrs:  1,
			errFields: []string{"args"},
		},
		{
			name:     "backend-config is valid",
			req:      &terraform.Request{Action: "init", Args: map[string]interface{}{"backend-config": "backend.hcl"}},
			wantErrs: 0,
		},
		{
			name:     "reconfigure bool true is valid",
			req:      &terraform.Request{Action: "init", Args: map[string]interface{}{"reconfigure": true}},
			wantErrs: 0,
		},
		{
			name:     "upgrade bool true is valid",
			req:      &terraform.Request{Action: "init", Args: map[string]interface{}{"upgrade": true}},
			wantErrs: 0,
		},
		{
			name:     "no-color bool true is valid",
			req:      &terraform.Request{Action: "init", Args: map[string]interface{}{"no-color": true}},
			wantErrs: 0,
		},
		{
			name:     "all known flags are valid",
			req:      &terraform.Request{Action: "init", Args: map[string]interface{}{"reconfigure": true, "upgrade": true, "no-color": true, "backend-config": "b.hcl"}},
			wantErrs: 0,
		},
		{
			name:      "unknown flag produces error",
			req:       &terraform.Request{Action: "init", Args: map[string]interface{}{"unknown-flag": "value"}},
			wantErrs:  1,
			errFields: []string{"unknown-flag"},
		},
		{
			name:     "multiple unknown flags each produce an error",
			req:      &terraform.Request{Action: "init", Args: map[string]interface{}{"bad1": true, "bad2": true}},
			wantErrs: 2,
		},
		{
			name:      "mix of valid and unknown flag",
			req:       &terraform.Request{Action: "init", Args: map[string]interface{}{"reconfigure": true, "bad-flag": true}},
			wantErrs:  1,
			errFields: []string{"bad-flag"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := cmd.Validate(tt.req)

			if tt.wantErrs == 0 && len(errs) != 0 {
				t.Fatalf("expected no errors, got %+v", errs)
			}
			if tt.wantErrs > 0 && len(errs) == 0 {
				t.Fatal("expected validation errors, got none")
			}
			if tt.wantErrs > 0 && len(errs) != tt.wantErrs {
				t.Errorf("expected %d errors, got %d: %+v", tt.wantErrs, len(errs), errs)
			}

			for _, field := range tt.errFields {
				if !containsField(errs, field) {
					t.Errorf("expected error on field %q, errors = %+v", field, errs)
				}
			}
		})
	}
}

// ── BuildArgs ─────────────────────────────────────────────────────────────────

func TestInitBuildArgs(t *testing.T) {
	cmd := &InitCommand{}

	tests := []struct {
		name       string
		req        *terraform.Request
		wantArgs   []string // every element must be present
		wantAbsent []string // none of these may be present
		firstArgIs string
	}{
		{
			name:       "first arg is always init",
			req:        &terraform.Request{Args: map[string]interface{}{"reconfigure": true}},
			firstArgIs: "init",
		},
		{
			name:     "backend-config flag",
			req:      &terraform.Request{Args: map[string]interface{}{"backend-config": "backend.hcl"}},
			wantArgs: []string{"init", "-backend-config=backend.hcl"},
		},
		{
			name:     "reconfigure bool",
			req:      &terraform.Request{Args: map[string]interface{}{"reconfigure": true}},
			wantArgs: []string{"-reconfigure"},
		},
		{
			name:     "upgrade bool",
			req:      &terraform.Request{Args: map[string]interface{}{"upgrade": true}},
			wantArgs: []string{"-upgrade"},
		},
		{
			name:     "no-color string true",
			req:      &terraform.Request{Args: map[string]interface{}{"no-color": "true"}},
			wantArgs: []string{"-no-color"},
		},
		{
			name:       "reconfigure false omits flag",
			req:        &terraform.Request{Args: map[string]interface{}{"reconfigure": false}},
			wantAbsent: []string{"-reconfigure"},
		},
		{
			name:     "all flags combined",
			req:      &terraform.Request{Args: map[string]interface{}{"backend-config": "b.hcl", "reconfigure": true, "upgrade": true, "no-color": true}},
			wantArgs: []string{"init", "-backend-config=b.hcl", "-reconfigure", "-upgrade", "-no-color"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := cmd.BuildArgs(tt.req)

			if tt.firstArgIs != "" && (len(args) == 0 || args[0] != tt.firstArgIs) {
				t.Errorf("BuildArgs()[0] = %q, want %q", safeFirst(args), tt.firstArgIs)
			}
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

func TestInitHelp_NonEmpty(t *testing.T) {
	cmd := &InitCommand{}
	if h := cmd.Help(); h == "" {
		t.Error("Help() should return non-empty string")
	}
}

func TestInitHelp_MentionsInit(t *testing.T) {
	cmd := &InitCommand{}
	if !strings.Contains(cmd.Help(), "init") {
		t.Error("Help() should mention 'init'")
	}
}
