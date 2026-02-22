package cli

import (
	"bytes"
	"testing"
)

func TestRootCmd_Version(t *testing.T) {
	cmd := newRootCmd("1.2.3")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	if got != "ancc version 1.2.3\n" {
		t.Errorf("got %q, want %q", got, "ancc version 1.2.3\n")
	}
}

func TestRootCmd_NoArgs(t *testing.T) {
	cmd := newRootCmd("dev")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	if len(got) == 0 {
		t.Error("expected help output, got empty string")
	}
}
