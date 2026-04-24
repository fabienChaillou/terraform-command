package contracts_test

import (
	"testing"

	"github.com/fabienChaillou/terraform-contracts/contracts"
)

func TestActionMap_Get_ExistingKey(t *testing.T) {
	m := contracts.ActionMap[int]{"apply": 1800}
	if got := m.Get("apply", 0); got != 1800 {
		t.Fatalf("Get(apply): want 1800, got %d", got)
	}
}

func TestActionMap_Get_MissingKey_ReturnsDefault(t *testing.T) {
	m := contracts.ActionMap[int]{"apply": 1800}
	if got := m.Get("init", 300); got != 300 {
		t.Fatalf("Get(missing): want 300, got %d", got)
	}
}

func TestActionMap_Get_EmptyMap_ReturnsDefault(t *testing.T) {
	var m contracts.ActionMap[string]
	if got := m.Get("plan", "fallback"); got != "fallback" {
		t.Fatalf("Get(empty): want fallback, got %q", got)
	}
}

func TestActionMap_Set_InitialisesNilMap(t *testing.T) {
	var m contracts.ActionMap[int]
	m.Set("workspace", 60)
	if got := m.Get("workspace", 0); got != 60 {
		t.Fatalf("Set then Get: want 60, got %d", got)
	}
}

func TestActionMap_Set_OverwritesExistingKey(t *testing.T) {
	m := contracts.ActionMap[int]{"plan": 900}
	m.Set("plan", 1200)
	if got := m.Get("plan", 0); got != 1200 {
		t.Fatalf("Set overwrite: want 1200, got %d", got)
	}
}

func TestActionMap_WorkflowNameRouting(t *testing.T) {
	routes := contracts.ActionMap[string]{
		"init":      contracts.WorkflowInit,
		"plan":      contracts.WorkflowPlan,
		"apply":     contracts.WorkflowApply,
		"destroy":   contracts.WorkflowDestroy,
		"workspace": contracts.WorkflowWorkspace,
	}

	cases := []struct {
		action string
		want   string
	}{
		{"init", contracts.WorkflowInit},
		{"plan", contracts.WorkflowPlan},
		{"apply", contracts.WorkflowApply},
		{"destroy", contracts.WorkflowDestroy},
		{"workspace", contracts.WorkflowWorkspace},
		{"unknown", contracts.WorkflowShellCommand}, // fallback
	}

	for _, tc := range cases {
		got := routes.Get(tc.action, contracts.WorkflowShellCommand)
		if got != tc.want {
			t.Errorf("action %q: want %q, got %q", tc.action, tc.want, got)
		}
	}
}
