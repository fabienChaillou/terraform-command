package command

import (
	"sort"
	"testing"
)

func TestRegistry_NewRegistry_ContainsAllCommands(t *testing.T) {
	r := NewRegistry()
	expected := []string{"apply", "destroy", "init", "plan", "workspace"}

	for _, name := range expected {
		if _, ok := r.Lookup(name); !ok {
			t.Errorf("NewRegistry() missing command %q", name)
		}
	}
}

func TestRegistry_Lookup_CaseInsensitive(t *testing.T) {
	r := NewRegistry()
	cases := []string{"PLAN", "Plan", "pLaN", "plan"}

	for _, name := range cases {
		if _, ok := r.Lookup(name); !ok {
			t.Errorf("Lookup(%q) should find command regardless of case", name)
		}
	}
}

func TestRegistry_Lookup_TrimsWhitespace(t *testing.T) {
	r := NewRegistry()
	cases := []string{" plan", "plan ", "  plan  ", "\tplan"}

	for _, name := range cases {
		if _, ok := r.Lookup(name); !ok {
			t.Errorf("Lookup(%q) should find command despite surrounding whitespace", name)
		}
	}
}

func TestRegistry_Lookup_UnknownAction(t *testing.T) {
	r := NewRegistry()
	unknown := []string{"", "fmt", "state", "validate", "unknown"}

	for _, name := range unknown {
		if _, ok := r.Lookup(name); ok {
			t.Errorf("Lookup(%q) should return false for unknown action", name)
		}
	}
}

func TestRegistry_Register_Override(t *testing.T) {
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

func TestRegistry_Register_NewCommand(t *testing.T) {
	r := NewRegistry()
	r.Register("custom", &InitCommand{})

	if _, ok := r.Lookup("custom"); !ok {
		t.Error("Lookup(custom) should find newly registered command")
	}
}

func TestRegistry_Actions_ReturnsAllNames(t *testing.T) {
	r := NewRegistry()
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
	r := NewRegistry()
	if h := r.GlobalHelp(); h == "" {
		t.Error("GlobalHelp() should return non-empty string")
	}
}
