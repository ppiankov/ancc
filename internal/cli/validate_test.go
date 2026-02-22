package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestValidateCmd_DefaultPath(t *testing.T) {
	cmd := newRootCmd("dev")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"validate"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, "validating .") {
		t.Errorf("expected default path '.', got %q", got)
	}
}

func TestValidateCmd_CustomPath(t *testing.T) {
	cmd := newRootCmd("dev")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"validate", "/tmp/somerepo"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, "validating /tmp/somerepo") {
		t.Errorf("expected custom path in output, got %q", got)
	}
}

func TestValidateCmd_Help(t *testing.T) {
	cmd := newRootCmd("dev")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"validate", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, "--format") {
		t.Error("expected --format in help output")
	}
	if !strings.Contains(got, "--verbose") {
		t.Error("expected --verbose in help output")
	}
}
