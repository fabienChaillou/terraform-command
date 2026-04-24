package contracts_test

import (
	"encoding/json"
	"testing"

	"github.com/fabienChaillou/terraform-contracts/contracts"
)

// ── ExecutionResult.Success() ─────────────────────────────────────────────────

func TestSuccess_ExitCodeZero(t *testing.T) {
	r := &contracts.ExecutionResult{ExitCode: 0}
	if !r.Success() {
		t.Error("Success() = false, want true for exit code 0")
	}
}

func TestSuccess_ExitCodeOne(t *testing.T) {
	r := &contracts.ExecutionResult{ExitCode: 1}
	if r.Success() {
		t.Error("Success() = true, want false for exit code 1")
	}
}

func TestSuccess_ExitCodeTwo(t *testing.T) {
	// Exit code 2 = "terraform plan detected changes" — not a success.
	r := &contracts.ExecutionResult{ExitCode: 2}
	if r.Success() {
		t.Error("Success() = true, want false for exit code 2 (plan changes)")
	}
}

func TestSuccess_NegativeExitCode(t *testing.T) {
	r := &contracts.ExecutionResult{ExitCode: -1}
	if r.Success() {
		t.Error("Success() = true, want false for negative exit code")
	}
}

// ── ExecutionResult JSON serialisation ───────────────────────────────────────

func TestExecutionResult_JSONRoundTrip(t *testing.T) {
	original := &contracts.ExecutionResult{
		ExitCode: 1,
		Stdout:   "Plan: 3 to add.",
		Stderr:   "Error: something failed",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var decoded contracts.ExecutionResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if decoded.ExitCode != original.ExitCode {
		t.Errorf("ExitCode: got %d, want %d", decoded.ExitCode, original.ExitCode)
	}
	if decoded.Stdout != original.Stdout {
		t.Errorf("Stdout: got %q, want %q", decoded.Stdout, original.Stdout)
	}
	if decoded.Stderr != original.Stderr {
		t.Errorf("Stderr: got %q, want %q", decoded.Stderr, original.Stderr)
	}
}

func TestExecutionResult_JSONFieldNames(t *testing.T) {
	r := &contracts.ExecutionResult{ExitCode: 2, Stdout: "out", Stderr: "err"}
	data, _ := json.Marshal(r)

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal to map: %v", err)
	}

	for _, key := range []string{"exit_code", "stdout", "stderr"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("JSON missing key %q (got: %v)", key, raw)
		}
	}
}

// ── ExecuteOptions ────────────────────────────────────────────────────────────

func TestExecuteOptions_ZeroValue(t *testing.T) {
	var opts contracts.ExecuteOptions
	if opts.Action != "" {
		t.Errorf("Action zero value: got %q, want empty string", opts.Action)
	}
	if opts.TimeoutSeconds != 0 {
		t.Errorf("TimeoutSeconds zero value: got %d, want 0", opts.TimeoutSeconds)
	}
	if opts.MaxAttempts != 0 {
		t.Errorf("MaxAttempts zero value: got %d, want 0", opts.MaxAttempts)
	}
}

func TestExecuteOptions_Fields(t *testing.T) {
	opts := contracts.ExecuteOptions{
		Action:         "apply",
		TimeoutSeconds: 1800,
		MaxAttempts:    1,
	}
	if opts.Action != "apply" {
		t.Errorf("Action: got %q, want %q", opts.Action, "apply")
	}
	if opts.TimeoutSeconds != 1800 {
		t.Errorf("TimeoutSeconds: got %d, want 1800", opts.TimeoutSeconds)
	}
	if opts.MaxAttempts != 1 {
		t.Errorf("MaxAttempts: got %d, want 1", opts.MaxAttempts)
	}
}
