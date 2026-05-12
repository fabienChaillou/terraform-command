package terraform

import "testing"

// ── ActionMap[T] ──────────────────────────────────────────────────────────────

func TestActionMap_Get_PresentKey(t *testing.T) {
	m := ActionMap[int]{"apply": 1800, "plan": 900}

	if got := m.Get("apply", 0); got != 1800 {
		t.Errorf("Get(apply) = %d, want 1800", got)
	}
	if got := m.Get("plan", 0); got != 900 {
		t.Errorf("Get(plan) = %d, want 900", got)
	}
}

func TestActionMap_Get_AbsentKeyReturnsDefault(t *testing.T) {
	m := ActionMap[int]{"apply": 1800}

	if got := m.Get("init", 300); got != 300 {
		t.Errorf("Get(init) = %d, want default 300", got)
	}
}

func TestActionMap_Get_NilMapReturnsDefault(t *testing.T) {
	var m ActionMap[int]

	if got := m.Get("plan", 42); got != 42 {
		t.Errorf("nil ActionMap.Get = %d, want 42", got)
	}
}

func TestActionMap_Get_ZeroDefault(t *testing.T) {
	m := ActionMap[int32]{}

	if got := m.Get("destroy", 0); got != 0 {
		t.Errorf("Get with zero default = %d, want 0", got)
	}
}

func TestActionMap_Get_StringType(t *testing.T) {
	m := ActionMap[string]{"env": "prod", "region": "eu-west-1"}

	if got := m.Get("env", "dev"); got != "prod" {
		t.Errorf("Get(env) = %q, want %q", got, "prod")
	}
	if got := m.Get("unknown", "dev"); got != "dev" {
		t.Errorf("Get(unknown) = %q, want default %q", got, "dev")
	}
}

func TestActionMap_Set_InitialisesNilMap(t *testing.T) {
	var m ActionMap[int]
	m.Set("plan", 900)

	if got := m.Get("plan", 0); got != 900 {
		t.Errorf("after Set, Get(plan) = %d, want 900", got)
	}
}

func TestActionMap_Set_OverwritesExistingKey(t *testing.T) {
	m := ActionMap[int]{"plan": 900}
	m.Set("plan", 600)

	if got := m.Get("plan", 0); got != 600 {
		t.Errorf("after Set override, Get(plan) = %d, want 600", got)
	}
}
