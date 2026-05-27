package command

import (
	"strings"
	"testing"

	"github.com/fabienChaillou/terraform-commander/terraform"
)

// taint_test.go covers both TaintCommand and UntaintCommand because they
// share validation and BuildArgs logic — exercising both proves the
// shared helpers also work through the untaint entry point.

// ── Validate ──────────────────────────────────────────────────────────────────

func TestTaintValidate(t *testing.T) {
	cases := []struct {
		name      string
		req       *terraform.Request
		wantErrs  int
		errFields []string
	}{
		{
			name:      "missing address",
			req:       &terraform.Request{Action: "taint", Args: map[string]interface{}{}},
			wantErrs:  1,
			errFields: []string{"address"},
		},
		{
			name:     "valid address",
			req:      &terraform.Request{Action: "taint", Args: map[string]interface{}{"address": "aws_instance.web"}},
			wantErrs: 0,
		},
		{
			name:     "all flags valid",
			req:      &terraform.Request{Action: "taint", Args: map[string]interface{}{"address": "aws_instance.web", "allow-missing": true, "lock": false, "lock-timeout": "30s"}},
			wantErrs: 0,
		},
		{
			name:      "lock as non-bool/string",
			req:       &terraform.Request{Action: "taint", Args: map[string]interface{}{"address": "aws_instance.web", "lock": 1}},
			wantErrs:  1,
			errFields: []string{"lock"},
		},
		{
			name:      "lock-timeout empty",
			req:       &terraform.Request{Action: "taint", Args: map[string]interface{}{"address": "aws_instance.web", "lock-timeout": "  "}},
			wantErrs:  1,
			errFields: []string{"lock-timeout"},
		},
		{
			name:      "unknown flag",
			req:       &terraform.Request{Action: "taint", Args: map[string]interface{}{"address": "aws_instance.web", "force": true}},
			wantErrs:  1,
			errFields: []string{"force"},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assertErrors(t, (&TaintCommand{}).Validate(tt.req), tt.wantErrs, tt.errFields)
		})
	}
}

func TestUntaintValidate_SharesBehaviour(t *testing.T) {
	// Untaint should behave identically to taint regarding validation.
	cmd := &UntaintCommand{}

	t.Run("missing address", func(t *testing.T) {
		errs := cmd.Validate(&terraform.Request{Action: "untaint", Args: map[string]interface{}{}})
		assertErrors(t, errs, 1, []string{"address"})
	})
	t.Run("valid address", func(t *testing.T) {
		errs := cmd.Validate(&terraform.Request{Action: "untaint", Args: map[string]interface{}{"address": "aws_instance.web"}})
		assertErrors(t, errs, 0, nil)
	})
	t.Run("unknown flag", func(t *testing.T) {
		errs := cmd.Validate(&terraform.Request{Action: "untaint", Args: map[string]interface{}{"address": "aws_instance.web", "force": true}})
		assertErrors(t, errs, 1, []string{"force"})
	})
}

// ── BuildArgs ─────────────────────────────────────────────────────────────────

func TestTaintBuildArgs(t *testing.T) {
	cmd := &TaintCommand{}
	t.Run("address only", func(t *testing.T) {
		args := cmd.BuildArgs(&terraform.Request{Args: map[string]interface{}{"address": "aws_instance.web"}})
		if args[0] != "taint" {
			t.Errorf("first arg should be 'taint', got %v", args)
		}
		if args[len(args)-1] != "aws_instance.web" {
			t.Errorf("address should be the last arg, got %v", args)
		}
	})
	t.Run("with all flags", func(t *testing.T) {
		args := cmd.BuildArgs(&terraform.Request{Args: map[string]interface{}{
			"address": "aws_instance.web", "allow-missing": true,
			"lock": false, "lock-timeout": "30s",
		}})
		if !containsArg(args, "-allow-missing") {
			t.Errorf("expected -allow-missing in %v", args)
		}
		if !containsArg(args, "-lock=false") {
			t.Errorf("expected -lock=false in %v", args)
		}
		if !containsArg(args, "-lock-timeout=30s") {
			t.Errorf("expected -lock-timeout=30s in %v", args)
		}
	})
	t.Run("lock=true emits -lock=true", func(t *testing.T) {
		args := cmd.BuildArgs(&terraform.Request{Args: map[string]interface{}{"address": "x", "lock": true}})
		if !containsArg(args, "-lock=true") {
			t.Errorf("expected -lock=true in %v", args)
		}
	})
}

func TestUntaintBuildArgs(t *testing.T) {
	cmd := &UntaintCommand{}
	args := cmd.BuildArgs(&terraform.Request{Args: map[string]interface{}{"address": "aws_instance.web", "allow-missing": true}})
	if args[0] != "untaint" {
		t.Errorf("first arg should be 'untaint', got %v", args)
	}
	if !containsArg(args, "-allow-missing") || args[len(args)-1] != "aws_instance.web" {
		t.Errorf("expected -allow-missing flag and trailing address, got %v", args)
	}
}

// ── Help ──────────────────────────────────────────────────────────────────────

func TestTaintHelp_MentionsTaint(t *testing.T) {
	if !strings.Contains((&TaintCommand{}).Help(), "taint") {
		t.Error("TaintCommand.Help() should mention 'taint'")
	}
	if !strings.Contains((&UntaintCommand{}).Help(), "untaint") {
		t.Error("UntaintCommand.Help() should mention 'untaint'")
	}
}
