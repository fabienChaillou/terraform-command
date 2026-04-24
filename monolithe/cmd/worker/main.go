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
