// Command worker is the Temporal worker binary for terraform-commander-worker.
//
// All workflows (init, plan, apply, destroy, workspace) are registered on a
// single worker polling contracts.TaskQueue ("shell-command-queue").
//
// # Production note
//
// For stricter isolation of destructive operations (apply / destroy) you can
// split into two workers at any time without changing this package's code:
//   - Move ApplyWorkflow / DestroyWorkflow registrations to a second worker
//     polling contracts.ApplyTaskQueue ("terraform-apply-queue") with
//     concurrency = 1.
//   - Update the server's TaskQueueByAction map to route those actions to
//     contracts.ApplyTaskQueue.
//
// Usage:
//
//	./worker
//	./worker --temporal-host my-cluster:7233 --namespace production
package main

import (
	"log"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	temporalworker "github.com/fabienChaillou/terraform-commander-worker/temporal"
	"github.com/fabienChaillou/terraform-contracts/contracts"
)

// workerOptions holds runtime configuration for this binary.
// In production, load these from environment variables or a config file.
type workerOptions struct {
	TemporalHost      string
	TemporalNamespace string
	Concurrency       int
}

func main() {
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

	// ── Single worker — contracts.TaskQueue ("shell-command-queue") ───────────
	// All five Terraform actions plus the generic fallback are handled here.
	w := worker.New(c, contracts.TaskQueue, worker.Options{
		MaxConcurrentActivityExecutionSize:     opts.Concurrency,
		MaxConcurrentWorkflowTaskExecutionSize: opts.Concurrency,
	})

	// Register every workflow under its contract name string so the server can
	// call it without importing this repo.
	w.RegisterWorkflowWithOptions(
		temporalworker.InitWorkflow,
		workflow.RegisterOptions{Name: contracts.WorkflowInit},
	)
	w.RegisterWorkflowWithOptions(
		temporalworker.PlanWorkflow,
		workflow.RegisterOptions{Name: contracts.WorkflowPlan},
	)
	w.RegisterWorkflowWithOptions(
		temporalworker.ApplyWorkflow,
		workflow.RegisterOptions{Name: contracts.WorkflowApply},
	)
	w.RegisterWorkflowWithOptions(
		temporalworker.DestroyWorkflow,
		workflow.RegisterOptions{Name: contracts.WorkflowDestroy},
	)
	w.RegisterWorkflowWithOptions(
		temporalworker.WorkspaceWorkflow,
		workflow.RegisterOptions{Name: contracts.WorkflowWorkspace},
	)
	w.RegisterWorkflowWithOptions(
		temporalworker.ShellCommandWorkflow,
		workflow.RegisterOptions{Name: contracts.WorkflowShellCommand},
	)

	// Single shared activity — all workflows delegate shell execution here.
	w.RegisterActivity(temporalworker.ExecuteShellCommand)

	// ── Start ─────────────────────────────────────────────────────────────────
	log.Printf("worker: polling %q (concurrency=%d)", contracts.TaskQueue, opts.Concurrency)

	if err := w.Start(); err != nil {
		log.Fatalf("worker: failed to start: %v", err)
	}

	// Block until SIGINT / SIGTERM.
	<-worker.InterruptCh()

	log.Println("worker: graceful shutdown initiated")
	w.Stop()
	log.Println("worker: shutdown complete")
}
