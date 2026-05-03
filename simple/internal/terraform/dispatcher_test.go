package terraform_test

import (
	"testing"

	"github.com/fabienChaillou/terraform-cmd/internal/terraform"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

func newDispatcher() *terraform.Dispatcher {
	return terraform.NewDispatcher(terraform.NewRegistry())
}

func assertValid(t *testing.T, result terraform.CommandResult) {
	t.Helper()
	if !result.Valid {
		t.Fatalf("expected valid result, got errors: %v", result.Errors)
	}
}

func assertInvalid(t *testing.T, result terraform.CommandResult, expectedFields ...string) {
	t.Helper()
	if result.Valid {
		t.Fatal("expected invalid result, but got valid")
	}
	if len(expectedFields) == 0 {
		return
	}
	fieldSet := make(map[string]bool)
	for _, e := range result.Errors {
		fieldSet[e.Field] = true
	}
	for _, f := range expectedFields {
		if !fieldSet[f] {
			t.Errorf("expected error on field %q, got errors: %v", f, result.Errors)
		}
	}
}

func assertHasHelp(t *testing.T, result terraform.CommandResult) {
	t.Helper()
	if result.HelpText == "" {
		t.Fatal("expected HelpText to be non-empty on invalid result")
	}
}

// ─── Dispatcher ──────────────────────────────────────────────────────────────

func TestDispatcher_MissingAction(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{})
	assertInvalid(t, result, "action")
	assertHasHelp(t, result)
}

func TestDispatcher_UnknownAction(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{"action": "nuke"})
	assertInvalid(t, result, "action")
	assertHasHelp(t, result)
}

func TestDispatcher_ActionNotString(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{"action": 42})
	assertInvalid(t, result, "action")
}

// ─── Init ────────────────────────────────────────────────────────────────────

func TestInit_Minimal(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{"action": "init"})
	assertValid(t, result)
	if result.Command != "init" {
		t.Errorf("expected command=init, got %q", result.Command)
	}
}

func TestInit_WithOptions(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{
		"action":  "init",
		"backend": true,
		"upgrade": true,
		"dir":     "./infra",
	})
	assertValid(t, result)
	if result.Payload["dir"] != "./infra" {
		t.Errorf("expected dir=./infra")
	}
}

// ─── Validate ────────────────────────────────────────────────────────────────

func TestValidate_Minimal(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{"action": "validate"})
	assertValid(t, result)
}

func TestValidate_JSONOutput(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{"action": "validate", "json": true})
	assertValid(t, result)
	if result.Payload["json"] != true {
		t.Error("json flag not in payload")
	}
}

// ─── Plan ─────────────────────────────────────────────────────────────────────

func TestPlan_Minimal(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{"action": "plan"})
	assertValid(t, result)
}

func TestPlan_WithOut(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{"action": "plan", "out": "tfplan"})
	assertValid(t, result)
	if result.Payload["out"] != "tfplan" {
		t.Error("out not in payload")
	}
}

func TestPlan_WithTargets(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{
		"action": "plan",
		"target": []any{"aws_instance.web", "aws_s3_bucket.data"},
	})
	assertValid(t, result)
}

// ─── Apply ────────────────────────────────────────────────────────────────────

func TestApply_WithAutoApprove(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{"action": "apply", "auto_approve": true})
	assertValid(t, result)
}

func TestApply_InvalidParallelism(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{"action": "apply", "parallelism": float64(200)})
	assertInvalid(t, result, "parallelism")
	assertHasHelp(t, result)
}

func TestApply_ValidParallelism(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{"action": "apply", "parallelism": float64(5)})
	assertValid(t, result)
	if result.Payload["parallelism"] != 5 {
		t.Errorf("expected parallelism=5")
	}
}

// ─── Destroy ──────────────────────────────────────────────────────────────────

func TestDestroy_MissingAutoApprove(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{"action": "destroy"})
	assertInvalid(t, result, "auto_approve")
	assertHasHelp(t, result)
}

func TestDestroy_AutoApproveFalse(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{"action": "destroy", "auto_approve": false})
	assertInvalid(t, result, "auto_approve")
}

func TestDestroy_Valid(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{"action": "destroy", "auto_approve": true})
	assertValid(t, result)
}

// ─── Workspace ────────────────────────────────────────────────────────────────

func TestWorkspace_MissingSubCommand(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{"action": "workspace"})
	assertInvalid(t, result, "sub_command")
	assertHasHelp(t, result)
}

func TestWorkspace_UnknownSubCommand(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{"action": "workspace", "sub_command": "nuke"})
	assertInvalid(t, result, "sub_command")
}

func TestWorkspace_List(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{"action": "workspace", "sub_command": "list"})
	assertValid(t, result)
	if result.SubCommand != "list" {
		t.Errorf("expected SubCommand=list, got %q", result.SubCommand)
	}
}

func TestWorkspace_New_MissingName(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{"action": "workspace", "sub_command": "new"})
	assertInvalid(t, result, "name")
	assertHasHelp(t, result)
}

func TestWorkspace_New_Valid(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{"action": "workspace", "sub_command": "new", "name": "staging"})
	assertValid(t, result)
	if result.Payload["name"] != "staging" {
		t.Error("name not in payload")
	}
}

func TestWorkspace_Select_Valid(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{"action": "workspace", "sub_command": "select", "name": "prod"})
	assertValid(t, result)
}

func TestWorkspace_Delete_WithForce(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{"action": "workspace", "sub_command": "delete", "name": "old-ws", "force": true})
	assertValid(t, result)
	if result.Payload["force"] != true {
		t.Error("force not in payload")
	}
}

func TestWorkspace_Show(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{"action": "workspace", "sub_command": "show"})
	assertValid(t, result)
}

// ─── State ────────────────────────────────────────────────────────────────────

func TestState_MissingSubCommand(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{"action": "state"})
	assertInvalid(t, result, "sub_command")
}

func TestState_List(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{"action": "state", "sub_command": "list"})
	assertValid(t, result)
}

func TestState_Show_MissingAddress(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{"action": "state", "sub_command": "show"})
	assertInvalid(t, result, "address")
	assertHasHelp(t, result)
}

func TestState_Show_Valid(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{"action": "state", "sub_command": "show", "address": "aws_instance.web"})
	assertValid(t, result)
}

func TestState_Mv_MissingDestination(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{"action": "state", "sub_command": "mv", "address": "aws_instance.old"})
	assertInvalid(t, result, "destination")
}

func TestState_Mv_Valid(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{
		"action":      "state",
		"sub_command": "mv",
		"address":     "aws_instance.old",
		"destination": "aws_instance.new",
	})
	assertValid(t, result)
}

func TestState_Rm_MissingAddress(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{"action": "state", "sub_command": "rm"})
	assertInvalid(t, result, "address")
}

// ─── Import ───────────────────────────────────────────────────────────────────

func TestImport_MissingFields(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{"action": "import"})
	assertInvalid(t, result, "address", "id")
	assertHasHelp(t, result)
}

func TestImport_MissingID(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{"action": "import", "address": "aws_instance.foo"})
	assertInvalid(t, result, "id")
}

func TestImport_Valid(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{
		"action":  "import",
		"address": "aws_instance.web",
		"id":      "i-1234567890abcdef0",
	})
	assertValid(t, result)
}

// ─── Output ───────────────────────────────────────────────────────────────────

func TestOutput_Minimal(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{"action": "output"})
	assertValid(t, result)
}

func TestOutput_Named(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{"action": "output", "name": "vpc_id", "json": true})
	assertValid(t, result)
	if result.Payload["name"] != "vpc_id" {
		t.Error("name not in payload")
	}
}

// ─── Graph ────────────────────────────────────────────────────────────────────

func TestGraph_InvalidType(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{"action": "graph", "type": "unknown"})
	assertInvalid(t, result, "type")
}

func TestGraph_ValidType(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{"action": "graph", "type": "plan"})
	assertValid(t, result)
}

// ─── Fmt ──────────────────────────────────────────────────────────────────────

func TestFmt_Check(t *testing.T) {
	d := newDispatcher()
	result := d.Dispatch(map[string]any{"action": "fmt", "check": true, "recursive": true})
	assertValid(t, result)
	if result.Payload["check"] != true {
		t.Error("check flag not in payload")
	}
}

// ─── Command interface ────────────────────────────────────────────────────────

func TestAllCommands_HelpNotEmpty(t *testing.T) {
	registry := terraform.NewRegistry()
	for _, action := range registry.Actions() {
		cmd, _ := registry.Lookup(action)
		if cmd.Help() == "" {
			t.Errorf("command %q returned empty Help()", action)
		}
	}
}

func TestAllCommands_Name(t *testing.T) {
	registry := terraform.NewRegistry()
	for _, action := range registry.Actions() {
		cmd, _ := registry.Lookup(action)
		if cmd.Name() != action {
			t.Errorf("expected Name()=%q, got %q", action, cmd.Name())
		}
	}
}
