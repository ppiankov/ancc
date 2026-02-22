package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestInitCmd_Stub(t *testing.T) {
	cmd := newRootCmd("dev")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"init"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, "not yet implemented") {
		t.Errorf("expected stub message, got %q", got)
	}
}

func TestInitCmd_Help(t *testing.T) {
	cmd := newRootCmd("dev")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"init", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, "--name") {
		t.Error("expected --name in help output")
	}
	if !strings.Contains(got, "--force") {
		t.Error("expected --force in help output")
	}
}
