package command

import (
	"sort"
	"testing"

	"github.com/fabienChaillou/terraform-commander/internal/terraform"
)

// registry_test.go covers the preloaded NewRegistry() constructor defined in
// this package.  Tests of the generic Registry[C] machinery itself live in
// the parent terraform package (registry_test.go).

func TestNewRegistry_ContainsAllCommands(t *testing.T) {
	r := NewRegistry()
	expected := []string{
		"apply", "destroy", "init", "output", "plan",
		"refresh", "state", "taint", "unlock", "untaint", "workspace",
	}

	for _, name := range expected {
		if _, ok := r.Lookup(name); !ok {
			t.Errorf("NewRegistry() missing command %q", name)
		}
	}
}

func TestNewRegistry_Actions_ReturnsAllNames(t *testing.T) {
	r := NewRegistry()
	actions := r.Actions()
	sort.Strings(actions)

	expected := []string{
		"apply", "destroy", "init", "output", "plan",
		"refresh", "state", "taint", "unlock", "untaint", "workspace",
	}
	if len(actions) != len(expected) {
		t.Fatalf("Actions() returned %d names, want %d: %v", len(actions), len(expected), actions)
	}
	for i, a := range actions {
		if a != expected[i] {
			t.Errorf("Actions()[%d] = %q, want %q", i, a, expected[i])
		}
	}
}

func TestNewRegistry_Lookup_CaseInsensitive(t *testing.T) {
	r := NewRegistry()
	for _, name := range []string{"PLAN", "Plan", "pLaN", "plan"} {
		if _, ok := r.Lookup(name); !ok {
			t.Errorf("Lookup(%q) should find command regardless of case", name)
		}
	}
}

func TestNewRegistry_Register_Override(t *testing.T) {
	r := NewRegistry()
	custom := &InitCommand{} // re-use an existing type as a stand-in
	r.Register("plan", custom)

	cmd, ok := r.Lookup("plan")
	if !ok {
		t.Fatal("Lookup(plan) returned false after re-registration")
	}
	if cmd != custom {
		t.Error("Lookup(plan) did not return the overriding command")
	}
}

func TestNewRegistry_Register_NewCommand(t *testing.T) {
	r := NewRegistry()
	r.Register("custom", &InitCommand{})

	if _, ok := r.Lookup("custom"); !ok {
		t.Error("Lookup(custom) should find newly registered command")
	}
}

func TestNewRegistry_GlobalHelp_NonEmpty(t *testing.T) {
	r := NewRegistry()
	if h := r.GlobalHelp(); h == "" {
		t.Error("GlobalHelp() should return non-empty string")
	}
}

// Compile-time check that the preloaded registry satisfies api.Resolver.
func TestNewRegistry_SatisfiesResolverInterface(t *testing.T) {
	type resolverLike interface {
		Lookup(action string) (terraform.Command, bool)
		GlobalHelp() string
	}
	var _ resolverLike = NewRegistry()
}
