package main

import (
	"log"
	"net/http"
	"os"

	temporalclient "go.temporal.io/sdk/client"

	"github.com/fabienChaillou/terraform-cmd/api"
	"github.com/fabienChaillou/terraform-cmd/internal/terraform"
	"github.com/fabienChaillou/terraform-cmd/internal/worker"
)

func main() {
	// ── Temporal client ──────────────────────────────────────────────────────
	temporalHost := os.Getenv("TEMPORAL_HOST")
	if temporalHost == "" {
		temporalHost = "localhost:7233"
	}

	tc, err := temporalclient.Dial(temporalclient.Options{
		HostPort: temporalHost,
	})
	if err != nil {
		log.Fatalf("failed to connect to Temporal: %v", err)
	}
	defer tc.Close()

	// ── Terraform registry + dispatcher ──────────────────────────────────────
	registry := terraform.NewRegistry()
	dispatcher := terraform.NewDispatcher(registry)

	// ── Temporal worker (registers all workflows + the shared activity) ──────
	terraformBin := os.Getenv("TERRAFORM_BIN") // default: "terraform"
	factory := worker.NewWorkerFactory(tc, registry, terraformBin)
	if err := factory.Start(); err != nil {
		log.Fatalf("failed to start Temporal worker: %v", err)
	}
	defer factory.Stop()

	// ── Huma router ──────────────────────────────────────────────────────────
	// Routes:
	//   POST /terraform/command  — main endpoint
	//   GET  /openapi.json       — OpenAPI 3.1 spec (auto)
	//   GET  /docs               — Swagger UI (auto)
	executor := worker.NewExecutor(tc)
	router := api.NewRouter(dispatcher, executor)

	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	log.Printf("Terraform API listening on %s (Temporal: %s)", addr, temporalHost)
	log.Printf("OpenAPI spec → http://localhost%s/openapi.json", addr)
	log.Printf("Swagger UI   → http://localhost%s/docs", addr)

	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal(err)
	}
}
