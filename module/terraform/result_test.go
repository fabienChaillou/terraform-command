package terraform

import "testing"

func TestExecutionResult_Success_TrueOnZeroExit(t *testing.T) {
	r := &ExecutionResult{ExitCode: 0}
	if !r.Success() {
		t.Error("Success() = false on ExitCode 0, want true")
	}
}

func TestExecutionResult_Success_FalseOnNonZeroExit(t *testing.T) {
	for _, code := range []int{1, 2, 127, 255, -1} {
		r := &ExecutionResult{ExitCode: code}
		if r.Success() {
			t.Errorf("Success() = true on ExitCode %d, want false", code)
		}
	}
}
