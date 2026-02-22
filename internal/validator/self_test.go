package validator

import (
	"path/filepath"
	"runtime"
	"testing"
)

// TestSelfValidation runs ancc's validator against the ancc repo itself.
// Any regression in SKILL.md will break this test.
func TestSelfValidation(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(file), "..", "..")

	result, err := Validate(repoRoot)
	if err != nil {
		t.Fatalf("self-validation error: %v", err)
	}

	if result.Summary.Fail > 0 {
		for _, c := range result.Checks {
			if c.Status == StatusFail {
				t.Errorf("FAIL: %s — %s", c.Name, c.Message)
			}
		}
		t.Fatalf("self-validation failed: %d check(s) failed", result.Summary.Fail)
	}

	if result.Summary.Total != 11 {
		t.Errorf("expected 11 checks, got %d", result.Summary.Total)
	}

	// binary-release is expected to warn for local validation.
	expectedPass := 10
	if result.Summary.Pass != expectedPass {
		t.Errorf("expected %d pass, got %d", expectedPass, result.Summary.Pass)
	}
}
