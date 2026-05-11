## example to build the main function

server main.go
```go
package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/danielgtaylor/huma/v2/humacli"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.temporal.io/sdk/client"

	"github.com/fabienChaillou/terraform-commander/api"
	"github.com/fabienChaillou/terraform-commander/internal/terraform"
	"github.com/fabienChaillou/terraform-commander/internal/terraform/command"
	temporalworker "github.com/fabienChaillou/terraform-commander/temporal"
)

// Options are the CLI flags exposed by humacli / cobra.
//
//	./terraform-commander --help
//	./terraform-commander --port 9090 --temporal-host my-cluster:7233
type Options struct {
	Host              string `doc:"Bind address"                        default:"127.0.0.1"`
	Port              int    `doc:"HTTP port"             short:"p"      default:"8080"`
	TemporalHost      string `doc:"Temporal frontend host:port"         default:"localhost:7233"`
	TemporalNamespace string `doc:"Temporal namespace"                  default:"default"`
	DefaultTimeout    int    `doc:"Default activity timeout in seconds" default:"1800"`
}

func main() {
	cli := humacli.New(func(hooks humacli.Hooks, opts *Options) {
		// ── Temporal client ───────────────────────────────────────────────────
		temporalClient, err := client.Dial(client.Options{
			HostPort:  opts.TemporalHost,
			Namespace: opts.TemporalNamespace,
		})
		if err != nil {
			log.Fatalf("cannot connect to Temporal (%s): %v", opts.TemporalHost, err)
		}

		// ── Workflow routing ──────────────────────────────────────────────────
		// Every action is mapped to a dedicated workflow function.
		// ShellCommandWorkflow remains as a fallback for unknown/custom actions.
		workflowByAction := terraform.ActionMap[interface{}]{
			"init":      temporalworker.InitWorkflow,
			"plan":      temporalworker.PlanWorkflow,
			"apply":     temporalworker.ApplyWorkflow,
			"destroy":   temporalworker.DestroyWorkflow,
			"workspace": temporalworker.WorkspaceWorkflow,
		}

		executor := &temporalworker.TemporalExecutor{
			Client:           temporalClient,
			DefaultTimeout:   time.Duration(opts.DefaultTimeout) * time.Second,
			MaxAttempts:      1,
			WorkflowByAction: workflowByAction,
		}

		// ── Command registry + config ─────────────────────────────────────────
		registry := command.NewRegistry()

		// Per-action timeout overrides (apply/destroy inherit DefaultTimeout).
		apiCfg := &api.Config{
			TimeoutByAction: terraform.ActionMap[int]{
				"apply":   opts.DefaultTimeout,
				"destroy": opts.DefaultTimeout,
			},
		}

		// ── Chi + Huma ────────────────────────────────────────────────────────
		router := chi.NewMux()
		router.Use(middleware.Logger)
		router.Use(middleware.Recoverer)

		humaConfig := huma.DefaultConfig("Terraform Commander API", "1.0.0")
		humaConfig.Info.Description =
			"Validates Terraform commands server-side and executes them " +
				"via a Temporal worker."

		humaAPI := humachi.New(router, humaConfig)
		api.RegisterRoutes(humaAPI, registry, executor, apiCfg)

		// ── Lifecycle ─────────────────────────────────────────────────────────
		hooks.OnStart(func() {
			addr := fmt.Sprintf("%s:%d", opts.Host, opts.Port)
			log.Printf("server   http://%s", addr)
			log.Printf("openapi  http://%s/openapi.json", addr)
			log.Printf("docs     http://%s/docs", addr)
			log.Printf("temporal %s", opts.TemporalHost)

			if err := http.ListenAndServe(addr, router); err != nil {
				log.Fatalf("server error: %v", err)
			}
		})

		hooks.OnStop(func() {
			temporalClient.Close()
			log.Println("shutdown complete")
		})
	})

	cli.Run()
}

```

worker.go
```go
// Command worker is the Temporal worker binary.
//
// It polls two task queues:
//
//   - "shell-command-queue"  — generic queue for init, workspace, plan.
//   - "terraform-apply-queue" — restricted queue for apply and destroy.
//
// Running two queues on the same binary is the simplest setup for development.
// In production, you would typically run separate worker processes per queue so
// that destructive operations (apply / destroy) can be isolated on hardened,
// audited hosts with stricter IAM / network policies.
//
// Usage:
//
//	./worker --temporal-host localhost:7233
//	./worker --temporal-host my-cluster:7233 --namespace production
//	./worker --concurrency 10
package main

import (
	"log"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	temporalworker "github.com/fabienChaillou/terraform-commander/temporal"
)

// workerOptions holds CLI configuration for this binary.
type workerOptions struct {
	TemporalHost      string
	TemporalNamespace string
	Concurrency       int
}

func main() {
	// In production, load these from env vars, flags, or a config file.
	opts := workerOptions{
		TemporalHost:      "localhost:7233",
		TemporalNamespace: "default",
		Concurrency:       5,
	}

	// ── Temporal client ───────────────────────────────────────────────────────
	c, err := client.Dial(client.Options{
		HostPort:  opts.TemporalHost,
		Namespace: opts.TemporalNamespace,
	})
	if err != nil {
		log.Fatalf("worker: cannot connect to Temporal (%s): %v", opts.TemporalHost, err)
	}
	defer c.Close()

	// ── Standard worker — "shell-command-queue" ───────────────────────────────
	// Handles: init, workspace, plan (and any action without a dedicated workflow).
	standardWorker := worker.New(c, temporalworker.TaskQueue, worker.Options{
		MaxConcurrentActivityExecutionSize:     opts.Concurrency,
		MaxConcurrentWorkflowTaskExecutionSize: opts.Concurrency,
	})

	// All five standard actions have dedicated workflows.
	// ShellCommandWorkflow is kept as a fallback for custom/unknown actions.
	standardWorker.RegisterWorkflow(temporalworker.ShellCommandWorkflow)
	standardWorker.RegisterWorkflow(temporalworker.InitWorkflow)
	standardWorker.RegisterWorkflow(temporalworker.PlanWorkflow)
	standardWorker.RegisterWorkflow(temporalworker.WorkspaceWorkflow)

	// The activity is queue-agnostic — registered on every worker that
	// needs to run shell commands.
	standardWorker.RegisterActivity(temporalworker.ExecuteShellCommand)

	// ── Apply worker — "terraform-apply-queue" ────────────────────────────────
	// Handles: apply and destroy exclusively.
	// In production this would be a separate binary on a hardened host.
	applyWorker := worker.New(c, temporalworker.ApplyTaskQueue, worker.Options{
		// Limit to 1 concurrent destructive operation per worker process.
		MaxConcurrentActivityExecutionSize:     1,
		MaxConcurrentWorkflowTaskExecutionSize: opts.Concurrency,
	})

	applyWorker.RegisterWorkflow(temporalworker.ApplyWorkflow)
	applyWorker.RegisterWorkflow(temporalworker.DestroyWorkflow)
	applyWorker.RegisterActivity(temporalworker.ExecuteShellCommand)

	// ── Start both workers ────────────────────────────────────────────────────
	log.Printf("worker: polling %q (concurrency=%d)", temporalworker.TaskQueue, opts.Concurrency)
	log.Printf("worker: polling %q (concurrency=1 — apply/destroy)", temporalworker.ApplyTaskQueue)

	// Start both workers in the background; interrupt channel handles graceful drain.
	if err := standardWorker.Start(); err != nil {
		log.Fatalf("worker: standard worker failed to start: %v", err)
	}
	if err := applyWorker.Start(); err != nil {
		log.Fatalf("worker: apply worker failed to start: %v", err)
	}

	// Block until SIGINT / SIGTERM.
	<-worker.InterruptCh()

	log.Println("worker: graceful shutdown initiated")
	standardWorker.Stop()
	applyWorker.Stop()
	log.Println("worker: shutdown complete")
}
```
