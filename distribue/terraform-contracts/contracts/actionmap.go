package contracts

// ActionMap is a generic helper for keying configuration or routing data by
// Terraform action name ("init", "plan", "apply", …).
//
// It wraps a plain map so that absent keys return an explicit default value
// rather than Go's silent zero value — making lookup intent visible at the
// call site.
//
//	timeouts := contracts.ActionMap[int]{
//	    "apply":   1800,
//	    "destroy": 1800,
//	    "plan":    900,
//	}
//	t := timeouts.Get("init", 300) // → 300 (default)
type ActionMap[T any] map[string]T

// Get returns the value for action if present, otherwise defaultVal.
func (m ActionMap[T]) Get(action string, defaultVal T) T {
	if v, ok := m[action]; ok {
		return v
	}
	return defaultVal
}

// Set stores v under action, initialising the underlying map if needed.
func (m *ActionMap[T]) Set(action string, v T) {
	if *m == nil {
		*m = make(ActionMap[T])
	}
	(*m)[action] = v
}
