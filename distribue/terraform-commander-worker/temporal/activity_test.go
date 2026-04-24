// Tests for ExecuteShellCommand.
//
// The tests override the package-level binaryName variable so they can run
// without terraform installed.  They are NOT marked t.Parallel() because they
// share that package variable.
//
// binaryName is reset to TerraformBinary via t.Cleanup after each test.
package temporal

import (
	"context"
	"strings"
	"testing"
)

// setBinary overrides binaryName for the duration of the test and restores
// TerraformBinary when the test ends.
func setBinary(t *testing.T, binary string) {
	t.Helper()
	binaryName = binary
	t.Cleanup(func() { binaryName = TerraformBinary })
}

// ── TerraformBinary const ─────────────────────────────────────────────────────

func TestTerraformBinary_Isterraform(t *testing.T) {
	if TerraformBinary != "terraform" {
		t.Errorf("TerraformBinary = %q, want %q", TerraformBinary, "terraform")
	}
}

// ── Success path ──────────────────────────────────────────────────────────────

func TestExecuteShellCommand_Success_ExitCode(t *testing.T) {
	setBinary(t, "echo")

	result, err := ExecuteShellCommand(context.Background(), ActivityInput{Args: []string{"hello"}})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
}

func TestExecuteShellCommand_Success_StdoutCaptured(t *testing.T) {
	setBinary(t, "echo")

	result, err := ExecuteShellCommand(context.Background(), ActivityInput{Args: []string{"terraform-test-output"}})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if !strings.Contains(result.Stdout, "terraform-test-output") {
		t.Errorf("Stdout = %q, want it to contain %q", result.Stdout, "terraform-test-output")
	}
}

func TestExecuteShellCommand_Success_StderrEmpty(t *testing.T) {
	setBinary(t, "echo")

	result, err := ExecuteShellCommand(context.Background(), ActivityInput{Args: []string{"ok"}})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result.Stderr != "" {
		t.Errorf("Stderr = %q, want empty for echo", result.Stderr)
	}
}

// ── Non-zero exit path ────────────────────────────────────────────────────────

func TestExecuteShellCommand_NonZeroExit_IsNotGoError(t *testing.T) {
	// `false` exits with code 1 on all POSIX systems.
	setBinary(t, "false")

	result, err := ExecuteShellCommand(context.Background(), ActivityInput{Args: []string{}})
	if err != nil {
		// A non-zero exit must NOT be returned as a Go error.
		t.Fatalf("non-zero exit should not produce a Go error, got: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
	if result.ExitCode == 0 {
		t.Errorf("ExitCode = 0, want non-zero")
	}
}

func TestExecuteShellCommand_NonZeroExit_ExitCodePreserved(t *testing.T) {
	// Use `sh -c "exit N"` to get a specific exit code.
	setBinary(t, "sh")

	result, err := ExecuteShellCommand(context.Background(), ActivityInput{
		Args: []string{"-c", "exit 2"},
	})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result.ExitCode != 2 {
		t.Errorf("ExitCode = %d, want 2", result.ExitCode)
	}
}

// ── Infrastructure error path ─────────────────────────────────────────────────

func TestExecuteShellCommand_BinaryNotFound_ReturnsGoError(t *testing.T) {
	setBinary(t, "/nonexistent/binary/that/does/not/exist")

	result, err := ExecuteShellCommand(context.Background(), ActivityInput{Args: []string{}})
	if err == nil {
		t.Errorf("expected Go error for missing binary, got result: %+v", result)
	}
}

// ── Args passthrough ──────────────────────────────────────────────────────────

func TestExecuteShellCommand_MultipleArgs_AllForwarded(t *testing.T) {
	// `printf` echoes exactly what we give it, without a trailing newline.
	setBinary(t, "printf")

	result, err := ExecuteShellCommand(context.Background(), ActivityInput{
		Args: []string{"%s-%s", "foo", "bar"},
	})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result.Stdout != "foo-bar" {
		t.Errorf("Stdout = %q, want %q", result.Stdout, "foo-bar")
	}
}

// ── Empty args ────────────────────────────────────────────────────────────────

func TestExecuteShellCommand_EmptyArgs(t *testing.T) {
	// `true` succeeds with no args.
	setBinary(t, "true")

	result, err := ExecuteShellCommand(context.Background(), ActivityInput{Args: []string{}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
}
