package skillmd

import (
	"path/filepath"
	"runtime"
	"testing"
)

func testdataPath(name string) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", name)
}

func TestParseFile_Valid(t *testing.T) {
	sf, err := ParseFile(testdataPath("valid-skill.md"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sf.Name != "mytool" {
		t.Errorf("Name = %q, want %q", sf.Name, "mytool")
	}
	if sf.Description != "A tool that does something useful." {
		t.Errorf("Description = %q, want %q", sf.Description, "A tool that does something useful.")
	}

	// Required sections.
	for _, heading := range []string{SectionInstall, SectionCommands, SectionWhatNotDo, SectionParsingExamples} {
		if sf.Sections[heading] == nil {
			t.Errorf("missing section %q", heading)
		}
	}

	// Commands.
	if len(sf.Commands) != 4 {
		t.Fatalf("got %d commands, want 4", len(sf.Commands))
	}

	run := sf.Commands[0]
	if run.Name != "mytool run" {
		t.Errorf("Commands[0].Name = %q, want %q", run.Name, "mytool run")
	}
	if run.Desc != "Runs the main operation." {
		t.Errorf("Commands[0].Desc = %q, want %q", run.Desc, "Runs the main operation.")
	}
	if len(run.Flags) != 2 {
		t.Errorf("Commands[0] has %d flags, want 2", len(run.Flags))
	} else {
		if run.Flags[0].Name != "--format json" {
			t.Errorf("flag[0].Name = %q, want %q", run.Flags[0].Name, "--format json")
		}
	}
	if run.JSONOutput == "" {
		t.Error("Commands[0].JSONOutput is empty, want JSON block")
	}
	if len(run.ExitCodes) != 2 {
		t.Errorf("Commands[0] has %d exit codes, want 2", len(run.ExitCodes))
	} else {
		if run.ExitCodes[0].Code != 0 || run.ExitCodes[1].Code != 1 {
			t.Errorf("exit codes = %v, want 0 and 1", run.ExitCodes)
		}
	}
}

func TestParseFile_Minimal(t *testing.T) {
	sf, err := ParseFile(testdataPath("minimal-skill.md"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sf.Name != "minimal" {
		t.Errorf("Name = %q, want %q", sf.Name, "minimal")
	}
	if len(sf.Commands) != 1 {
		t.Fatalf("got %d commands, want 1", len(sf.Commands))
	}
	if sf.Commands[0].Name != "minimal run" {
		t.Errorf("command name = %q, want %q", sf.Commands[0].Name, "minimal run")
	}
	if len(sf.Commands[0].Flags) != 1 {
		t.Errorf("got %d flags, want 1", len(sf.Commands[0].Flags))
	}
}

func TestParseFile_MissingSections(t *testing.T) {
	sf, err := ParseFile(testdataPath("missing-sections.md"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sf.Name != "incomplete" {
		t.Errorf("Name = %q, want %q", sf.Name, "incomplete")
	}

	// Should have Install and Commands, but not WhatNotDo or ParsingExamples.
	if sf.Sections[SectionInstall] == nil {
		t.Error("expected Install section")
	}
	if sf.Sections[SectionCommands] == nil {
		t.Error("expected Commands section")
	}
	if sf.Sections[SectionWhatNotDo] != nil {
		t.Error("WhatNotDo section should be nil")
	}
	if sf.Sections[SectionParsingExamples] != nil {
		t.Error("ParsingExamples section should be nil")
	}

	// Command should exist but lack JSON output and exit codes.
	if len(sf.Commands) != 1 {
		t.Fatalf("got %d commands, want 1", len(sf.Commands))
	}
	cmd := sf.Commands[0]
	if cmd.JSONOutput != "" {
		t.Errorf("expected empty JSONOutput, got %q", cmd.JSONOutput)
	}
	if len(cmd.ExitCodes) != 0 {
		t.Errorf("expected 0 exit codes, got %d", len(cmd.ExitCodes))
	}
}

func TestParseFile_Malformed(t *testing.T) {
	sf, err := ParseFile(testdataPath("malformed-skill.md"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No H1 — name should be empty.
	if sf.Name != "" {
		t.Errorf("Name = %q, want empty", sf.Name)
	}

	// Sections still parsed by H2.
	if sf.Sections[SectionCommands] == nil {
		t.Error("expected Commands section even if malformed")
	}
	if sf.Sections[SectionInstall] == nil {
		t.Error("expected Install section")
	}

	// No H3 in Commands — no commands extracted.
	if len(sf.Commands) != 0 {
		t.Errorf("got %d commands, want 0", len(sf.Commands))
	}
}

func TestParseFile_NotFound(t *testing.T) {
	_, err := ParseFile(testdataPath("nonexistent.md"))
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestParse_EmptyInput(t *testing.T) {
	sf, err := Parse("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sf.Name != "" {
		t.Errorf("Name = %q, want empty", sf.Name)
	}
	if len(sf.Sections) != 0 {
		t.Errorf("got %d sections, want 0", len(sf.Sections))
	}
	if len(sf.Commands) != 0 {
		t.Errorf("got %d commands, want 0", len(sf.Commands))
	}
}

func TestParse_FlagParsing(t *testing.T) {
	input := `# tool

Desc.

## Commands

### tool cmd

Does stuff.

**Flags:**
- ` + "`--format json`" + ` — JSON output
- ` + "`--verbose`" + ` — verbose mode
`
	sf, err := Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sf.Commands) != 1 {
		t.Fatalf("got %d commands, want 1", len(sf.Commands))
	}
	flags := sf.Commands[0].Flags
	if len(flags) != 2 {
		t.Fatalf("got %d flags, want 2", len(flags))
	}
	if flags[0].Name != "--format json" {
		t.Errorf("flag[0].Name = %q, want %q", flags[0].Name, "--format json")
	}
	if flags[1].Name != "--verbose" {
		t.Errorf("flag[1].Name = %q, want %q", flags[1].Name, "--verbose")
	}
	if flags[1].Desc != "verbose mode" {
		t.Errorf("flag[1].Desc = %q, want %q", flags[1].Desc, "verbose mode")
	}
}

func TestParse_ExitCodeParsing(t *testing.T) {
	input := `# tool

Desc.

## Commands

### tool cmd

Does stuff.

**Exit codes:**
- 0: all good
- 1: something failed
- 2: partial
`
	sf, err := Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	codes := sf.Commands[0].ExitCodes
	if len(codes) != 3 {
		t.Fatalf("got %d exit codes, want 3", len(codes))
	}
	if codes[2].Code != 2 || codes[2].Desc != "partial" {
		t.Errorf("exit code[2] = %+v, want {2, partial}", codes[2])
	}
}

func TestParse_MultipleCommands(t *testing.T) {
	input := `# tool

Desc.

## Commands

### tool alpha

First command.

### tool beta

Second command.

### tool gamma

Third command.
`
	sf, err := Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sf.Commands) != 3 {
		t.Fatalf("got %d commands, want 3", len(sf.Commands))
	}
	names := []string{sf.Commands[0].Name, sf.Commands[1].Name, sf.Commands[2].Name}
	want := []string{"tool alpha", "tool beta", "tool gamma"}
	for i := range names {
		if names[i] != want[i] {
			t.Errorf("command[%d] = %q, want %q", i, names[i], want[i])
		}
	}
}
