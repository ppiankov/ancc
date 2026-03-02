package skills

import "testing"

func TestContextWindow_Known(t *testing.T) {
	if got := ContextWindow(AgentClaudeCode); got != 165_000 {
		t.Errorf("ContextWindow(%q) = %d, want 165000", AgentClaudeCode, got)
	}
	if got := ContextWindow(AgentCline); got != 128_000 {
		t.Errorf("ContextWindow(%q) = %d, want 128000", AgentCline, got)
	}
}

func TestContextWindow_Unknown(t *testing.T) {
	if got := ContextWindow("unknown-agent"); got != 128_000 {
		t.Errorf("ContextWindow(%q) = %d, want 128000", "unknown-agent", got)
	}
}
