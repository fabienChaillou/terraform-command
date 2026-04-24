package command

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

// ── Registry[C] generics ──────────────────────────────────────────────────────

// TestRegistryOf_TypedRegistry verifies that NewRegistryOf[C] creates a typed
// registry constrained to C.  We use a simple inner interface to exercise the
// generic constraint beyond the base Command.
func TestRegistryOf_EmptyOnCreation(t *testing.T) {
	r := NewRegistryOf[Command]()
	if actions := r.Actions(); len(actions) != 0 {
		t.Errorf("NewRegistryOf should be empty, got %v", actions)
	}
}

func TestRegistryOf_RegisterAndLookup(t *testing.T) {
	r := NewRegistryOf[Command]()
	cmd := &PlanCommand{}
	r.Register("plan", cmd)

	got, ok := r.Lookup("plan")
	if !ok {
		t.Fatal("Lookup(plan) returned false")
	}
	if got != cmd {
		t.Error("Lookup(plan) returned wrong command")
	}
}

func TestNewRegistry_IsPreloaded(t *testing.T) {
	r := NewRegistry() // *Registry[Command]
	for _, action := range []string{"init", "plan", "apply", "destroy", "workspace"} {
		if _, ok := r.Lookup(action); !ok {
			t.Errorf("NewRegistry() missing %q", action)
		}
	}
}

// TestRegistry_SatisfiesResolver verifies at compile time that
// *Registry[Command] satisfies the api.Resolver-like interface
// (Lookup + GlobalHelp).  We define the interface inline to avoid
// importing the api package from command tests.
func TestRegistry_SatisfiesResolverInterface(t *testing.T) {
	type resolverLike interface {
		Lookup(action string) (Command, bool)
		GlobalHelp() string
	}
	var _ resolverLike = NewRegistry() // compile-time check
}
