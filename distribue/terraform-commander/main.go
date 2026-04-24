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
	"github.com/fabienChaillou/terraform-commander/command"
	temporalworker "github.com/fabienChaillou/terraform-commander/temporal"
	"github.com/fabienChaillou/terraform-contracts/contracts"
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

		// ── Workflow routing (by NAME, not function reference) ────────────────
		//
		// The server never imports the worker repo.  Instead it uses the string
		// constants from terraform-contracts; the worker registers its functions
		// under those exact names.  Adding a new action only requires updating
		// the worker repo + contracts — this file stays unchanged.
		workflowByAction := contracts.ActionMap[string]{
			"init":      contracts.WorkflowInit,
			"plan":      contracts.WorkflowPlan,
			"apply":     contracts.WorkflowApply,
			"destroy":   contracts.WorkflowDestroy,
			"workspace": contracts.WorkflowWorkspace,
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
			TimeoutByAction: command.ActionMap[int]{
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
