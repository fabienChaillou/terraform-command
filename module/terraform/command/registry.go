package command

import (
	"github.com/fabienChaillou/terraform-commander/internal/terraform"
)

// NewRegistry returns a *terraform.Registry[terraform.Command] preloaded with
// all standard Terraform sub-commands: init, plan, apply, destroy, workspace.
//
// It is defined here — in the command sub-package — rather than in the parent
// terraform package because it has to reference the concrete command types
// (InitCommand, PlanCommand, …), which would otherwise create an import cycle
// (terraform → command → terraform).
//
// Composition root usage:
//
//	registry := command.NewRegistry()
//	api.RegisterRoutes(humaAPI, registry, executor, cfg)
func NewRegistry() *terraform.Registry[terraform.Command] {
	r := terraform.NewRegistryOf[terraform.Command]()
	r.Register("init", &InitCommand{})
	r.Register("plan", &PlanCommand{})
	r.Register("apply", &ApplyCommand{})
	r.Register("destroy", &DestroyCommand{})
	r.Register("workspace", &WorkspaceCommand{})
	return r
}
