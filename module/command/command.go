// Package command is deprecated.
//
// All contents have moved:
//
//   - Command, Validator, ArgBuilder, HelpProvider, Request, ValidationError →
//     github.com/fabienChaillou/terraform-commander/internal/terraform
//   - Registry[C], NewRegistryOf →
//     github.com/fabienChaillou/terraform-commander/internal/terraform
//   - ExecutionResult, ExecuteOptions, ActionMap →
//     github.com/fabienChaillou/terraform-commander/internal/terraform
//   - InitCommand, PlanCommand, ApplyCommand, DestroyCommand, WorkspaceCommand,
//     NewRegistry →
//     github.com/fabienChaillou/terraform-commander/internal/terraform/command
//
// This file is left only because the surrounding filesystem prevents the
// directory from being removed in place.  Delete the whole "command/"
// directory from your checkout once the refactor lands:
//
//	git rm -rf command/
package command
