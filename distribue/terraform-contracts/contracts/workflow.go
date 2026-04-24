package contracts

// Task queue names used by Temporal workers.
//
// The server submits workflows to these queues; the worker polls them.
// Keeping them here ensures both repos use the same string literals.
const (
	// TaskQueue is the standard task queue for read-only / idempotent actions.
	// Handles: init, plan, workspace, and any unknown/custom actions.
	TaskQueue = "shell-command-queue"

	// ApplyTaskQueue is the restricted task queue for destructive actions.
	// Handles: apply and destroy exclusively.
	// In production, workers polling this queue run on hardened, audited hosts
	// with stricter IAM and network policies, concurrency = 1.
	ApplyTaskQueue = "terraform-apply-queue"
)

// Workflow name constants.
//
// The server calls workflows by name (string), not by function reference,
// so it never needs to import the worker repo.  Workers register their
// workflow functions under these exact names via workflow.RegisterOptions.
//
//	w.RegisterWorkflowWithOptions(InitWorkflow, workflow.RegisterOptions{
//	    Name: contracts.WorkflowInit,
//	})
const (
	WorkflowInit         = "InitWorkflow"
	WorkflowPlan         = "PlanWorkflow"
	WorkflowApply        = "ApplyWorkflow"
	WorkflowDestroy      = "DestroyWorkflow"
	WorkflowWorkspace    = "WorkspaceWorkflow"
	WorkflowShellCommand = "ShellCommandWorkflow" // fallback for unknown actions
)

// WorkflowInput is the single argument passed to every Terraform workflow.
//
// It is serialised to JSON by the Temporal client and deserialised by the
// worker, so both repos must agree on this structure — hence it lives here.
//
// The binary (terraform) is intentionally absent: it is a compile-time
// constant fixed by the worker's activity package (TerraformBinary).
type WorkflowInput struct {
	// Args are the CLI arguments passed to terraform, e.g.
	// ["init", "-backend=true"] or ["apply", "-auto-approve", "-var", "k=v"].
	Args []string `json:"args"`

	// TimeoutSeconds is the maximum wall-clock duration for the activity.
	// 0 means "use the workflow's built-in default".
	TimeoutSeconds int `json:"timeout_seconds,omitempty"`

	// MaxAttempts is the maximum number of activity attempts.
	// Destructive workflows (apply, destroy) override this to 1 internally.
	MaxAttempts int32 `json:"max_attempts,omitempty"`
}

// TaskQueueForAction returns the correct Temporal task queue for the given
// terraform action, using the provided ActionMap for lookups.
//
// This helper lives in contracts so both server and worker can use consistent
// routing logic without duplicating the map definition.
func TaskQueueForAction(action string, overrides ActionMap[string]) string {
	// Apply and destroy always go to the restricted queue by default.
	defaults := ActionMap[string]{
		"apply":   ApplyTaskQueue,
		"destroy": ApplyTaskQueue,
	}
	// Check caller overrides first, then built-in defaults, then standard queue.
	if q := overrides.Get(action, ""); q != "" {
		return q
	}
	return defaults.Get(action, TaskQueue)
}
