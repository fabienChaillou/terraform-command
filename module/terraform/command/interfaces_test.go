package command

import (
	"github.com/fabienChaillou/terraform-commander/internal/terraform"
)

// interfaces_test.go contains compile-time interface satisfaction checks.
//
// These tests produce no runtime output — they exist solely to ensure that
// the concrete commands satisfy every interface in the ISP hierarchy.
// A compilation failure here means an interface was broken.

// ── Compile-time checks for every concrete command ────────────────────────────

var (
	// Each concrete command must satisfy the three focused interfaces individually.
	_ terraform.Validator    = (*InitCommand)(nil)
	_ terraform.ArgBuilder   = (*InitCommand)(nil)
	_ terraform.HelpProvider = (*InitCommand)(nil)
	_ terraform.Command      = (*InitCommand)(nil) // composite

	_ terraform.Validator    = (*PlanCommand)(nil)
	_ terraform.ArgBuilder   = (*PlanCommand)(nil)
	_ terraform.HelpProvider = (*PlanCommand)(nil)
	_ terraform.Command      = (*PlanCommand)(nil)

	_ terraform.Validator    = (*ApplyCommand)(nil)
	_ terraform.ArgBuilder   = (*ApplyCommand)(nil)
	_ terraform.HelpProvider = (*ApplyCommand)(nil)
	_ terraform.Command      = (*ApplyCommand)(nil)

	_ terraform.Validator    = (*DestroyCommand)(nil)
	_ terraform.ArgBuilder   = (*DestroyCommand)(nil)
	_ terraform.HelpProvider = (*DestroyCommand)(nil)
	_ terraform.Command      = (*DestroyCommand)(nil)

	_ terraform.Validator    = (*WorkspaceCommand)(nil)
	_ terraform.ArgBuilder   = (*WorkspaceCommand)(nil)
	_ terraform.HelpProvider = (*WorkspaceCommand)(nil)
	_ terraform.Command      = (*WorkspaceCommand)(nil)
)
