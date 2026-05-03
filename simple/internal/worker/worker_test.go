package worker_test

import (
	"testing"

	"github.com/fabienChaillou/terraform-cmd/internal/terraform"
	"github.com/fabienChaillou/terraform-cmd/internal/worker"
)

// ─── WorkflowName ────────────────────────────────────────────────────────────

func TestWorkflowName(t *testing.T) {
	cases := []struct {
		action string
		want   string
	}{
		{"init", "InitWorkflow"},
		{"plan", "PlanWorkflow"},
		{"apply", "ApplyWorkflow"},
		{"destroy", "DestroyWorkflow"},
		{"validate", "ValidateWorkflow"},
		{"workspace", "WorkspaceWorkflow"},
		{"state", "StateWorkflow"},
		{"output", "OutputWorkflow"},
		{"import", "ImportWorkflow"},
		{"show", "ShowWorkflow"},
		{"fmt", "FmtWorkflow"},
		{"refresh", "RefreshWorkflow"},
		{"providers", "ProvidersWorkflow"},
		{"graph", "GraphWorkflow"},
		{"", "UnknownWorkflow"},
	}

	for _, tc := range cases {
		t.Run(tc.action, func(t *testing.T) {
			got := worker.WorkflowName(tc.action)
			if got != tc.want {
				t.Errorf("WorkflowName(%q) = %q, want %q", tc.action, got, tc.want)
			}
		})
	}
}

// ─── BuildArgs helpers ────────────────────────────────────────────────────────

func makeResult(command, sub string, payload map[string]any) terraform.CommandResult {
	return terraform.CommandResult{
		Command:    command,
		SubCommand: sub,
		Valid:      true,
		Payload:    payload,
	}
}

func containsArg(args []string, arg string) bool {
	for _, a := range args {
		if a == arg {
			return true
		}
	}
	return false
}

func firstArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

// ─── BuildArgs: init ─────────────────────────────────────────────────────────

func TestBuildArgs_Init_Minimal(t *testing.T) {
	r := makeResult("init", "", map[string]any{})
	args := worker.BuildArgs(r)
	if firstArg(args) != "init" {
		t.Errorf("expected first arg 'init', got %v", args)
	}
}

func TestBuildArgs_Init_AllFlags(t *testing.T) {
	r := makeResult("init", "", map[string]any{
		"backend":        true,
		"upgrade":        true,
		"reconfigure":    true,
		"migrate_state":  true,
		"backend_config": "backend.hcl",
		"dir":            "./infra",
	})
	args := worker.BuildArgs(r)
	for _, expected := range []string{"-backend=true", "-upgrade", "-reconfigure", "-migrate-state", "-backend-config=backend.hcl", "./infra"} {
		if !containsArg(args, expected) {
			t.Errorf("expected arg %q in %v", expected, args)
		}
	}
}

// ─── BuildArgs: plan ─────────────────────────────────────────────────────────

func TestBuildArgs_Plan_WithOut(t *testing.T) {
	r := makeResult("plan", "", map[string]any{"out": "tfplan"})
	args := worker.BuildArgs(r)
	if !containsArg(args, "-out=tfplan") {
		t.Errorf("expected -out=tfplan in %v", args)
	}
}

func TestBuildArgs_Plan_WithTargets(t *testing.T) {
	r := makeResult("plan", "", map[string]any{
		"target": []string{"aws_instance.web", "aws_s3_bucket.data"},
	})
	args := worker.BuildArgs(r)
	if !containsArg(args, "-target=aws_instance.web") {
		t.Errorf("missing -target=aws_instance.web in %v", args)
	}
	if !containsArg(args, "-target=aws_s3_bucket.data") {
		t.Errorf("missing -target=aws_s3_bucket.data in %v", args)
	}
}

func TestBuildArgs_Plan_WithVars(t *testing.T) {
	r := makeResult("plan", "", map[string]any{
		"var": []string{"env=prod", "region=eu-west-1"},
	})
	args := worker.BuildArgs(r)
	if !containsArg(args, "-var=env=prod") {
		t.Errorf("missing -var=env=prod in %v", args)
	}
	if !containsArg(args, "-var=region=eu-west-1") {
		t.Errorf("missing -var=region=eu-west-1 in %v", args)
	}
}

func TestBuildArgs_Plan_Destroy(t *testing.T) {
	r := makeResult("plan", "", map[string]any{"destroy": true, "out": "destroy.plan"})
	args := worker.BuildArgs(r)
	if !containsArg(args, "-destroy") {
		t.Errorf("expected -destroy in %v", args)
	}
}

// ─── BuildArgs: apply ─────────────────────────────────────────────────────────

func TestBuildArgs_Apply_AutoApprove(t *testing.T) {
	r := makeResult("apply", "", map[string]any{"auto_approve": true})
	args := worker.BuildArgs(r)
	if !containsArg(args, "-auto-approve") {
		t.Errorf("expected -auto-approve in %v", args)
	}
}

func TestBuildArgs_Apply_Parallelism(t *testing.T) {
	r := makeResult("apply", "", map[string]any{"parallelism": 5})
	args := worker.BuildArgs(r)
	if !containsArg(args, "-parallelism=5") {
		t.Errorf("expected -parallelism=5 in %v", args)
	}
}

func TestBuildArgs_Apply_PlanFile(t *testing.T) {
	r := makeResult("apply", "", map[string]any{"plan": "tfplan"})
	args := worker.BuildArgs(r)
	if !containsArg(args, "-plan=tfplan") {
		t.Errorf("expected -plan=tfplan in %v", args)
	}
}

// ─── BuildArgs: destroy ───────────────────────────────────────────────────────

func TestBuildArgs_Destroy_AutoApprove(t *testing.T) {
	r := makeResult("destroy", "", map[string]any{"auto_approve": true})
	args := worker.BuildArgs(r)
	if firstArg(args) != "destroy" {
		t.Errorf("expected 'destroy' first, got %v", args)
	}
	if !containsArg(args, "-auto-approve") {
		t.Errorf("expected -auto-approve in %v", args)
	}
}

// ─── BuildArgs: workspace ─────────────────────────────────────────────────────

func TestBuildArgs_Workspace_List(t *testing.T) {
	r := makeResult("workspace", "list", map[string]any{"sub_command": "list"})
	args := worker.BuildArgs(r)
	if len(args) < 2 || args[0] != "workspace" || args[1] != "list" {
		t.Errorf("expected [workspace list ...], got %v", args)
	}
}

func TestBuildArgs_Workspace_New(t *testing.T) {
	r := makeResult("workspace", "new", map[string]any{
		"sub_command": "new",
		"name":        "staging",
	})
	args := worker.BuildArgs(r)
	if args[0] != "workspace" || args[1] != "new" {
		t.Errorf("expected [workspace new ...], got %v", args)
	}
	if !containsArg(args, "staging") {
		t.Errorf("expected 'staging' in %v", args)
	}
}

func TestBuildArgs_Workspace_Delete_Force(t *testing.T) {
	r := makeResult("workspace", "delete", map[string]any{
		"sub_command": "delete",
		"name":        "old-ws",
		"force":       true,
	})
	args := worker.BuildArgs(r)
	if !containsArg(args, "-force") {
		t.Errorf("expected -force in %v", args)
	}
	if !containsArg(args, "old-ws") {
		t.Errorf("expected 'old-ws' in %v", args)
	}
}

// ─── BuildArgs: state ─────────────────────────────────────────────────────────

func TestBuildArgs_State_Show(t *testing.T) {
	r := makeResult("state", "show", map[string]any{
		"sub_command": "show",
		"address":     "aws_instance.web",
	})
	args := worker.BuildArgs(r)
	if args[0] != "state" || args[1] != "show" {
		t.Errorf("expected [state show ...], got %v", args)
	}
	if !containsArg(args, "aws_instance.web") {
		t.Errorf("expected address in %v", args)
	}
}

func TestBuildArgs_State_Mv(t *testing.T) {
	r := makeResult("state", "mv", map[string]any{
		"sub_command": "mv",
		"address":     "aws_instance.old",
		"destination": "aws_instance.new",
	})
	args := worker.BuildArgs(r)
	if !containsArg(args, "aws_instance.old") {
		t.Errorf("expected address in %v", args)
	}
	if !containsArg(args, "aws_instance.new") {
		t.Errorf("expected destination in %v", args)
	}
}

// ─── BuildArgs: import ────────────────────────────────────────────────────────

func TestBuildArgs_Import(t *testing.T) {
	r := makeResult("import", "", map[string]any{
		"address": "aws_instance.web",
		"id":      "i-1234567890abcdef0",
	})
	args := worker.BuildArgs(r)
	if firstArg(args) != "import" {
		t.Errorf("expected 'import' first, got %v", args)
	}
	if !containsArg(args, "aws_instance.web") {
		t.Errorf("expected address in %v", args)
	}
	if !containsArg(args, "i-1234567890abcdef0") {
		t.Errorf("expected id in %v", args)
	}
}

// ─── BuildArgs: fmt ───────────────────────────────────────────────────────────

func TestBuildArgs_Fmt_Check(t *testing.T) {
	r := makeResult("fmt", "", map[string]any{
		"check":     true,
		"recursive": true,
		"diff":      true,
	})
	args := worker.BuildArgs(r)
	for _, flag := range []string{"-check", "-recursive", "-diff"} {
		if !containsArg(args, flag) {
			t.Errorf("expected %q in %v", flag, args)
		}
	}
}

// ─── BuildArgs: output ────────────────────────────────────────────────────────

func TestBuildArgs_Output_JSON(t *testing.T) {
	r := makeResult("output", "", map[string]any{"json": true, "name": "vpc_id"})
	args := worker.BuildArgs(r)
	if !containsArg(args, "-json") {
		t.Errorf("expected -json in %v", args)
	}
	if !containsArg(args, "vpc_id") {
		t.Errorf("expected name in %v", args)
	}
}

// ─── BuildArgs: graph ─────────────────────────────────────────────────────────

func TestBuildArgs_Graph_Type(t *testing.T) {
	r := makeResult("graph", "", map[string]any{"type": "plan", "draw_cycles": true})
	args := worker.BuildArgs(r)
	if !containsArg(args, "-type=plan") {
		t.Errorf("expected -type=plan in %v", args)
	}
	if !containsArg(args, "-draw-cycles") {
		t.Errorf("expected -draw-cycles in %v", args)
	}
}

// ─── WorkflowName covers all registry actions ─────────────────────────────────

func TestWorkflowName_AllRegistryActions(t *testing.T) {
	registry := terraform.NewRegistry()
	for _, action := range registry.Actions() {
		name := worker.WorkflowName(action)
		if name == "" || name == "UnknownWorkflow" {
			t.Errorf("WorkflowName(%q) returned unexpected %q", action, name)
		}
	}
}
