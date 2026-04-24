package command

// interfaces_test.go contains compile-time interface satisfaction checks.
//
// These tests produce no runtime output — they exist solely to ensure that
// the concrete commands satisfy every interface in the ISP hierarchy.
// A compilation failure here means an interface was broken.

// ── Compile-time checks for every concrete command ────────────────────────────

var (
	// Each concrete command must satisfy the three focused interfaces individually.
	_ Validator    = (*InitCommand)(nil)
	_ ArgBuilder   = (*InitCommand)(nil)
	_ HelpProvider = (*InitCommand)(nil)
	_ Command      = (*InitCommand)(nil) // composite

	_ Validator    = (*PlanCommand)(nil)
	_ ArgBuilder   = (*PlanCommand)(nil)
	_ HelpProvider = (*PlanCommand)(nil)
	_ Command      = (*PlanCommand)(nil)

	_ Validator    = (*ApplyCommand)(nil)
	_ ArgBuilder   = (*ApplyCommand)(nil)
	_ HelpProvider = (*ApplyCommand)(nil)
	_ Command      = (*ApplyCommand)(nil)

	_ Validator    = (*DestroyCommand)(nil)
	_ ArgBuilder   = (*DestroyCommand)(nil)
	_ HelpProvider = (*DestroyCommand)(nil)
	_ Command      = (*DestroyCommand)(nil)

	_ Validator    = (*WorkspaceCommand)(nil)
	_ ArgBuilder   = (*WorkspaceCommand)(nil)
	_ HelpProvider = (*WorkspaceCommand)(nil)
	_ Command      = (*WorkspaceCommand)(nil)
)
