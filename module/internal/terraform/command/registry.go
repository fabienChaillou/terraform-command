package command

import (
	"github.com/fabienChaillou/terraform-commander/internal/terraform"
)

// NewRegistry returns a *terraform.Registry[terraform.Command] preloaded with
// every Terraform sub-command implemented in this package:
//
//	init, plan, apply, destroy, workspace,
//	state, taint, untaint, unlock, refresh, output
//
// It is defined here — in the command sub-package — rather than in the parent
// terraform package because it has to reference the concrete command types
// (InitCommand, PlanCommand, …), which would otherwise create an import cycle
// (terraform → command → terraform).
//
// Note: the "unlock" action maps to the CLI verb "force-unlock"; the action
// name is kept short for HTTP callers — see unlock.go for details.
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
	r.Register("state", &StateCommand{})
	r.Register("taint", &TaintCommand{})
	r.Register("untaint", &UntaintCommand{})
	r.Register("unlock", &UnlockCommand{}) // CLI verb: force-unlock
	r.Register("refresh", &RefreshCommand{})
	r.Register("output", &OutputCommand{})
	return r
}
