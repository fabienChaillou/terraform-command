# Refactor: monolithic `command/` → `internal/terraform` + `internal/terraform/command`

## Resulting layout

```
monolithe/
├── api/                                # uses internal/terraform (types only)
├── cmd/worker/                         # unchanged (no command-pkg dep)
├── command/                            # ⚠️ legacy — delete this folder
├── internal/
│   └── terraform/                      # package terraform: contracts + registry
│       ├── command.go                  # Command, Validator, ArgBuilder, HelpProvider,
│       │                               # Request, ValidationError
│       ├── registry.go                 # Registry[C], NewRegistryOf, GlobalHelp
│       ├── result.go                   # ExecutionResult, ExecuteOptions, ActionMap[T]
│       └── *_test.go
│       └── command/                    # package command: concrete commands
│           ├── init.go, plan.go, apply.go, destroy.go, workspace.go
│           ├── state.go, taint.go, untaint.go, unlock.go,
│           │   refresh.go, output.go   # NEW commands
│           ├── validation.go           # private helpers (unknownFlagErrors, …)
│           ├── registry.go             # NewRegistry() preloaded with the 11 commands
│           └── *_test.go               # one file per command + interfaces_test.go
├── temporal/                           # uses internal/terraform
├── go.mod, go.sum, main.go
```

## Import map

| Before                                                | After                                                                |
| ----------------------------------------------------- | -------------------------------------------------------------------- |
| `github.com/.../command.Command`                      | `github.com/.../internal/terraform.Command`                          |
| `command.Request`, `command.ValidationError`          | `terraform.Request`, `terraform.ValidationError`                     |
| `command.ExecutionResult`, `command.ExecuteOptions`   | `terraform.ExecutionResult`, `terraform.ExecuteOptions`              |
| `command.ActionMap`                                   | `terraform.ActionMap`                                                |
| `command.NewRegistry()`                               | `command.NewRegistry()` (now in `internal/terraform/command`)        |
| `command.InitCommand`, `command.PlanCommand`, …       | `command.InitCommand`, …  (same name, new path)                      |

## Cycle-free split

`internal/terraform/command` imports `internal/terraform`.  The parent
`terraform` package never imports back.  The previously-monolithic
`NewRegistry()` (which instantiated concrete commands) had to move into the
`command` sub-package so that import direction stays one-way.

## New commands added

| Action       | CLI verb produced  | Required args        | Notable flags                                  |
| ------------ | ------------------ | -------------------- | ---------------------------------------------- |
| `state`      | `state <sub>`      | depends on subcmd    | subcommands: list, show, mv, rm, pull, push    |
| `taint`      | `taint`            | `address`            | `allow-missing`, `lock`, `lock-timeout`        |
| `untaint`    | `untaint`          | `address`            | same flag set as taint                         |
| `unlock`     | `force-unlock`     | `lock-id`            | `force`                                        |
| `refresh`    | `refresh`          | —                    | `var`, `var-file`, `target`, `lock-timeout`, `no-color` |
| `output`     | `output`           | — (`name` if `raw`)  | `json`, `raw`, `state`, `no-color`             |

`unlock` is the API-side action name; the actual Terraform CLI verb is
`force-unlock`.  This is documented in `unlock.go`.

The new commands fall back to `temporalworker.ShellCommandWorkflow` because
no dedicated workflow is wired in `main.go`'s `WorkflowByAction` map.
Add per-command workflows there if you want command-specific timeout /
retry policy.

## ⚠️  Legacy `command/` folder

The sandbox couldn't delete the original `command/` directory in place, so
every file in it has been overwritten with a `package command` stub plus a
deprecation note in `command/command.go`.  The package still compiles to an
empty unit, so `go build ./...` keeps working.

**Recommended cleanup before committing:**

```bash
git rm -rf command/
```

## Running the tests

```bash
go test ./internal/terraform/...
```

This runs both the `terraform` package tests (generic Registry, ActionMap,
ExecutionResult) and the `terraform/command` package tests (per-command
Validate / BuildArgs / Help, plus compile-time interface assertions).
