package terraform

import (
	"sort"
	"testing"
)

// ── Generic Registry[C] tests (no concrete commands) ──────────────────────────
//
// Tests that rely on the preloaded NewRegistry() factory — which depends on
// the concrete InitCommand, PlanCommand, etc. — live in the command sub-package
// (registry_test.go in internal/terraform/command).

// stubCommand is a minimal Command implementation used to exercise the
// generic registry without importing the command sub-package (which would
// create a circular dependency).
type stubCommand struct{ name string }

func (s *stubCommand) Validate(_ *Request) []ValidationError { return nil }
func (s *stubCommand) BuildArgs(_ *Request) []string         { return []string{s.name} }
func (s *stubCommand) Help() string                          { return "stub: " + s.name }

func TestRegistryOf_EmptyOnCreation(t *testing.T) {
	r := NewRegistryOf[Command]()
	if actions := r.Actions(); len(actions) != 0 {
		t.Errorf("NewRegistryOf should be empty, got %v", actions)
	}
}

func TestRegistry_RegisterAndLookup(t *testing.T) {
	r := NewRegistryOf[Command]()
	cmd := &stubCommand{name: "stub"}
	r.Register("stub", cmd)

	got, ok := r.Lookup("stub")
	if !ok {
		t.Fatal("Lookup(stub) returned false")
	}
	if got != cmd {
		t.Error("Lookup(stub) returned wrong command")
	}
}

func TestRegistry_Lookup_CaseInsensitive(t *testing.T) {
	r := NewRegistryOf[Command]()
	r.Register("plan", &stubCommand{name: "plan"})

	for _, name := range []string{"PLAN", "Plan", "pLaN", "plan"} {
		if _, ok := r.Lookup(name); !ok {
			t.Errorf("Lookup(%q) should find command regardless of case", name)
		}
	}
}

func TestRegistry_Lookup_TrimsWhitespace(t *testing.T) {
	r := NewRegistryOf[Command]()
	r.Register("plan", &stubCommand{name: "plan"})

	for _, name := range []string{" plan", "plan ", "  plan  ", "\tplan"} {
		if _, ok := r.Lookup(name); !ok {
			t.Errorf("Lookup(%q) should find command despite surrounding whitespace", name)
		}
	}
}

func TestRegistry_Lookup_UnknownAction(t *testing.T) {
	r := NewRegistryOf[Command]()
	r.Register("plan", &stubCommand{name: "plan"})

	for _, name := range []string{"", "fmt", "state", "validate", "unknown"} {
		if _, ok := r.Lookup(name); ok {
			t.Errorf("Lookup(%q) should return false for unknown action", name)
		}
	}
}

func TestRegistry_Register_Override(t *testing.T) {
	r := NewRegistryOf[Command]()
	first := &stubCommand{name: "first"}
	second := &stubCommand{name: "second"}
	r.Register("plan", first)
	r.Register("plan", second)

	cmd, ok := r.Lookup("plan")
	if !ok {
		t.Fatal("Lookup(plan) returned false after re-registration")
	}
	if cmd != second {
		t.Error("Lookup(plan) did not return the overriding command")
	}
}

func TestRegistry_Actions_ReturnsAllNames(t *testing.T) {
	r := NewRegistryOf[Command]()
	for _, name := range []string{"apply", "destroy", "init", "plan", "workspace"} {
		r.Register(name, &stubCommand{name: name})
	}

	actions := r.Actions()
	sort.Strings(actions)

	expected := []string{"apply", "destroy", "init", "plan", "workspace"}
	if len(actions) != len(expected) {
		t.Fatalf("Actions() returned %d names, want %d: %v", len(actions), len(expected), actions)
	}
	for i, a := range actions {
		if a != expected[i] {
			t.Errorf("Actions()[%d] = %q, want %q", i, a, expected[i])
		}
	}
}

func TestRegistry_GlobalHelp_NonEmpty(t *testing.T) {
	r := NewRegistryOf[Command]()
	if h := r.GlobalHelp(); h == "" {
		t.Error("GlobalHelp() should return non-empty string")
	}
}

// Compile-time assertion that *Registry[Command] satisfies a Resolver-like
// interface (Lookup + GlobalHelp) — mirrors what the api package needs.
func TestRegistry_SatisfiesResolverInterface(t *testing.T) {
	type resolverLike interface {
		Lookup(action string) (Command, bool)
		GlobalHelp() string
	}
	var _ resolverLike = NewRegistryOf[Command]() // compile-time check
}
