package validator

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ppiankov/ancc/internal/skillmd"
)

func testdataPath(name string) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", name)
}

func loadFixture(t *testing.T, name string) *skillmd.SkillFile {
	t.Helper()
	sf, err := skillmd.ParseFile(testdataPath(name))
	if err != nil {
		t.Fatalf("failed to load fixture %s: %v", name, err)
	}
	return sf
}

// --- Individual check tests ---

func TestCheckSkillMDExists_Present(t *testing.T) {
	// testdata dir doesn't have SKILL.md, but repo root does.
	_, file, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(file), "..", "..")
	r := checkSkillMDExists(repoRoot)
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q", r.Status, StatusPass)
	}
}

func TestCheckSkillMDExists_DocsSubdir(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(filepath.Join(docsDir, "SKILL.md"), []byte("# test")); err != nil {
		t.Fatal(err)
	}
	r := checkSkillMDExists(dir)
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q", r.Status, StatusPass)
	}
}

func TestCheckSkillMDExists_Missing(t *testing.T) {
	r := checkSkillMDExists(t.TempDir())
	if r.Status != StatusFail {
		t.Errorf("status = %q, want %q", r.Status, StatusFail)
	}
}

func TestCheckInstall_Present(t *testing.T) {
	sf := loadFixture(t, "valid-skill.md")
	r := checkInstall(sf)
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q", r.Status, StatusPass)
	}
}

func TestCheckInstall_Missing(t *testing.T) {
	sf := &skillmd.SkillFile{Sections: map[string]*skillmd.Section{}}
	r := checkInstall(sf)
	if r.Status != StatusFail {
		t.Errorf("status = %q, want %q", r.Status, StatusFail)
	}
}

func TestCheckCommands_Valid(t *testing.T) {
	sf := loadFixture(t, "valid-skill.md")
	r := checkCommands(sf)
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q", r.Status, StatusPass)
	}
}

func TestCheckCommands_NoSection(t *testing.T) {
	sf := &skillmd.SkillFile{Sections: map[string]*skillmd.Section{}}
	r := checkCommands(sf)
	if r.Status != StatusFail {
		t.Errorf("status = %q, want %q", r.Status, StatusFail)
	}
}

func TestCheckCommands_EmptySection(t *testing.T) {
	sf := &skillmd.SkillFile{
		Sections: map[string]*skillmd.Section{
			skillmd.SectionCommands: {Heading: "Commands", Content: "no subheadings"},
		},
	}
	r := checkCommands(sf)
	if r.Status != StatusFail {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusFail, r.Message)
	}
}

func TestCheckFlags_Present(t *testing.T) {
	sf := loadFixture(t, "valid-skill.md")
	r := checkFlags(sf)
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q", r.Status, StatusPass)
	}
}

func TestCheckFlags_Missing(t *testing.T) {
	sf := loadFixture(t, "missing-sections.md")
	r := checkFlags(sf)
	if r.Status != StatusFail {
		t.Errorf("status = %q, want %q", r.Status, StatusFail)
	}
}

func TestCheckJSONOutput_Present(t *testing.T) {
	sf := loadFixture(t, "valid-skill.md")
	r := checkJSONOutput(sf)
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q", r.Status, StatusPass)
	}
}

func TestCheckJSONOutput_Missing(t *testing.T) {
	sf := loadFixture(t, "missing-sections.md")
	r := checkJSONOutput(sf)
	if r.Status != StatusFail {
		t.Errorf("status = %q, want %q", r.Status, StatusFail)
	}
}

func TestCheckExitCodes_Present(t *testing.T) {
	sf := loadFixture(t, "valid-skill.md")
	r := checkExitCodes(sf)
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q", r.Status, StatusPass)
	}
}

func TestCheckExitCodes_Missing(t *testing.T) {
	sf := loadFixture(t, "missing-sections.md")
	r := checkExitCodes(sf)
	if r.Status != StatusFail {
		t.Errorf("status = %q, want %q", r.Status, StatusFail)
	}
}

func TestCheckNotDo_Present(t *testing.T) {
	sf := loadFixture(t, "valid-skill.md")
	r := checkNotDo(sf)
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q", r.Status, StatusPass)
	}
}

func TestCheckNotDo_Missing(t *testing.T) {
	sf := loadFixture(t, "missing-sections.md")
	r := checkNotDo(sf)
	if r.Status != StatusFail {
		t.Errorf("status = %q, want %q", r.Status, StatusFail)
	}
}

func TestCheckParsing_Present(t *testing.T) {
	sf := loadFixture(t, "valid-skill.md")
	r := checkParsing(sf)
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q", r.Status, StatusPass)
	}
}

func TestCheckParsing_Missing(t *testing.T) {
	sf := loadFixture(t, "missing-sections.md")
	r := checkParsing(sf)
	if r.Status != StatusFail {
		t.Errorf("status = %q, want %q", r.Status, StatusFail)
	}
}

func TestCheckInitCommand_Present(t *testing.T) {
	sf := loadFixture(t, "valid-skill.md")
	r := checkInitCommand(sf)
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q", r.Status, StatusPass)
	}
}

func TestCheckInitCommand_Missing(t *testing.T) {
	sf := loadFixture(t, "missing-sections.md")
	r := checkInitCommand(sf)
	if r.Status != StatusFail {
		t.Errorf("status = %q, want %q", r.Status, StatusFail)
	}
}

func TestCheckDoctorCommand_Present(t *testing.T) {
	sf := loadFixture(t, "valid-skill.md")
	r := checkDoctorCommand(sf)
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q", r.Status, StatusPass)
	}
}

func TestCheckDoctorCommand_Missing(t *testing.T) {
	sf := loadFixture(t, "missing-sections.md")
	r := checkDoctorCommand(sf)
	if r.Status != StatusWarn {
		t.Errorf("status = %q, want %q (warn, not fail)", r.Status, StatusWarn)
	}
}

func TestCheckBinaryRelease_Skipped(t *testing.T) {
	r := checkBinaryRelease("/some/local/path")
	if r.Status != StatusWarn {
		t.Errorf("status = %q, want %q", r.Status, StatusWarn)
	}
}

// --- Orchestrator tests ---

func TestValidate_ValidFixture(t *testing.T) {
	// Create a temp dir with a copy of valid-skill.md as SKILL.md.
	dir := t.TempDir()
	sf, err := skillmd.ParseFile(testdataPath("valid-skill.md"))
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	// We need a real SKILL.md file in the temp dir.
	// Re-read raw content and write it.
	data, err := readFile(testdataPath("valid-skill.md"))
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}
	if err := writeFile(filepath.Join(dir, "SKILL.md"), data); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}
	_ = sf

	result, err := Validate(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 20 checks now (15 original + 5 semantic)
	if result.Summary.Total != 20 {
		t.Errorf("total = %d, want 20", result.Summary.Total)
	}
	if result.Summary.Fail != 0 {
		t.Errorf("fail = %d, want 0", result.Summary.Fail)
	}
	// binary-release is the only warn; semantic checks pass when no relevant sections
	if result.Summary.Warn != 1 {
		t.Errorf("warn = %d, want 1 (binary-release skipped)", result.Summary.Warn)
	}
	if result.Status != OverallPartial {
		t.Errorf("status = %q, want %q", result.Status, OverallPartial)
	}
}

func TestValidate_MissingSkillMD(t *testing.T) {
	dir := t.TempDir()
	result, err := Validate(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != OverallFail {
		t.Errorf("status = %q, want %q", result.Status, OverallFail)
	}
	if result.Summary.Total != 20 {
		t.Errorf("total = %d, want 20", result.Summary.Total)
	}
}

func TestValidate_MissingSections(t *testing.T) {
	dir := t.TempDir()
	data, err := readFile(testdataPath("missing-sections.md"))
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}
	if err := writeFile(filepath.Join(dir, "SKILL.md"), data); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}

	result, err := Validate(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != OverallFail {
		t.Errorf("status = %q, want %q", result.Status, OverallFail)
	}
	if result.Summary.Fail == 0 {
		t.Error("expected at least one failure")
	}
}

func TestValidate_DocsSubdir(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := readFile(testdataPath("valid-skill.md"))
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}
	if err := writeFile(filepath.Join(docsDir, "SKILL.md"), data); err != nil {
		t.Fatalf("failed to write docs/SKILL.md: %v", err)
	}

	result, err := Validate(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Summary.Total != 20 {
		t.Errorf("total = %d, want 20", result.Summary.Total)
	}
	if result.Summary.Fail != 0 {
		t.Errorf("fail = %d, want 0", result.Summary.Fail)
	}
}

func TestComputeSummary_AllPass(t *testing.T) {
	r := &ValidationResult{
		Checks: []CheckResult{
			{Name: "a", Status: StatusPass},
			{Name: "b", Status: StatusPass},
		},
	}
	computeSummary(r)
	if r.Status != OverallPass {
		t.Errorf("status = %q, want %q", r.Status, OverallPass)
	}
	if r.Summary.Pass != 2 {
		t.Errorf("pass = %d, want 2", r.Summary.Pass)
	}
}

func TestComputeSummary_WarnOnly(t *testing.T) {
	r := &ValidationResult{
		Checks: []CheckResult{
			{Name: "a", Status: StatusPass},
			{Name: "b", Status: StatusWarn},
		},
	}
	computeSummary(r)
	if r.Status != OverallPartial {
		t.Errorf("status = %q, want %q", r.Status, OverallPartial)
	}
}

func TestComputeSummary_WithFail(t *testing.T) {
	r := &ValidationResult{
		Checks: []CheckResult{
			{Name: "a", Status: StatusPass},
			{Name: "b", Status: StatusWarn},
			{Name: "c", Status: StatusFail},
		},
	}
	computeSummary(r)
	if r.Status != OverallFail {
		t.Errorf("status = %q, want %q", r.Status, OverallFail)
	}
	if r.Summary.Total != 3 {
		t.Errorf("total = %d, want 3", r.Summary.Total)
	}
}

// --- Semantic quality check tests ---

func TestCheckJSONExamplesValid_AllValid(t *testing.T) {
	sf := loadFixture(t, "valid-skill.md")
	r := checkJSONExamplesValid(sf)
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusPass, r.Message)
	}
}

func TestCheckJSONExamplesValid_NoJSON(t *testing.T) {
	sf := &skillmd.SkillFile{
		Commands: []skillmd.Command{{Name: "tool run"}},
	}
	r := checkJSONExamplesValid(sf)
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q", r.Status, StatusPass)
	}
}

func TestCheckJSONExamplesValid_InvalidJSON(t *testing.T) {
	sf := &skillmd.SkillFile{
		Commands: []skillmd.Command{
			{Name: "tool run", JSONOutput: `{"broken": true,}`},
		},
	}
	r := checkJSONExamplesValid(sf)
	if r.Status != StatusWarn {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusWarn, r.Message)
	}
}

func TestCheckExitCodesNumeric_HasZero(t *testing.T) {
	sf := loadFixture(t, "valid-skill.md")
	r := checkExitCodesNumeric(sf)
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusPass, r.Message)
	}
}

func TestCheckExitCodesNumeric_NoExitCodes(t *testing.T) {
	sf := &skillmd.SkillFile{
		Commands: []skillmd.Command{{Name: "tool run"}},
	}
	r := checkExitCodesNumeric(sf)
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q", r.Status, StatusPass)
	}
}

func TestCheckExitCodesNumeric_MissingZero(t *testing.T) {
	sf := &skillmd.SkillFile{
		Commands: []skillmd.Command{
			{Name: "tool run", ExitCodes: []skillmd.ExitCode{{Code: 1, Desc: "error"}}},
		},
	}
	r := checkExitCodesNumeric(sf)
	if r.Status != StatusWarn {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusWarn, r.Message)
	}
}

func TestCheckCommandsNotPlaceholder_Clean(t *testing.T) {
	sf := loadFixture(t, "valid-skill.md")
	r := checkCommandsNotPlaceholder(sf)
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusPass, r.Message)
	}
}

func TestCheckCommandsNotPlaceholder_PlaceholderName(t *testing.T) {
	sf := &skillmd.SkillFile{
		Name:     "mytool",
		Commands: []skillmd.Command{{Name: "mytool run"}},
	}
	r := checkCommandsNotPlaceholder(sf)
	if r.Status != StatusWarn {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusWarn, r.Message)
	}
}

func TestCheckCommandsNotPlaceholder_TemplateVar(t *testing.T) {
	sf := &skillmd.SkillFile{
		Name:     "realtool",
		Commands: []skillmd.Command{{Name: "<tool> run"}},
	}
	r := checkCommandsNotPlaceholder(sf)
	if r.Status != StatusWarn {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusWarn, r.Message)
	}
}

func TestCheckInstallHasCommand_Present(t *testing.T) {
	sf := loadFixture(t, "valid-skill.md")
	r := checkInstallHasCommand(sf)
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusPass, r.Message)
	}
}

func TestCheckInstallHasCommand_NoSection(t *testing.T) {
	sf := &skillmd.SkillFile{Sections: map[string]*skillmd.Section{}}
	r := checkInstallHasCommand(sf)
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q (no section to check)", r.Status, StatusPass)
	}
}

func TestCheckInstallHasCommand_NoCommand(t *testing.T) {
	sf := &skillmd.SkillFile{
		Sections: map[string]*skillmd.Section{
			skillmd.SectionInstall: {
				Heading: "Install",
				Content: "Download from the website and follow instructions.",
			},
		},
	}
	r := checkInstallHasCommand(sf)
	if r.Status != StatusWarn {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusWarn, r.Message)
	}
}

func TestCheckInstallHasCommand_GoInstall(t *testing.T) {
	sf := &skillmd.SkillFile{
		Sections: map[string]*skillmd.Section{
			skillmd.SectionInstall: {
				Heading: "Install",
				Content: "```\ngo install github.com/example/tool@latest\n```",
			},
		},
	}
	r := checkInstallHasCommand(sf)
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusPass, r.Message)
	}
}

// --- Semantic quality checks for agent SKILL.md files ---

func TestCheckTriggersActionable_Valid(t *testing.T) {
	sf := loadFixture(t, "valid-agent-skill.md")
	r := checkTriggersActionable(sf)
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusPass, r.Message)
	}
}

func TestCheckTriggersActionable_Vague(t *testing.T) {
	sf := loadFixture(t, "vague-skill.md")
	r := checkTriggersActionable(sf)
	if r.Status != StatusWarn {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusWarn, r.Message)
	}
}

func TestCheckTriggersActionable_NoSection(t *testing.T) {
	sf := &skillmd.SkillFile{Sections: map[string]*skillmd.Section{}}
	r := checkTriggersActionable(sf)
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q (no trigger section)", r.Status, StatusPass)
	}
}

func TestCheckToolsReferenceReal_Valid(t *testing.T) {
	sf := loadFixture(t, "valid-agent-skill.md")
	r := checkToolsReferenceReal(sf)
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusPass, r.Message)
	}
}

func TestCheckToolsReferenceReal_Suspicious(t *testing.T) {
	sf := loadFixture(t, "vague-skill.md")
	r := checkToolsReferenceReal(sf)
	if r.Status != StatusWarn {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusWarn, r.Message)
	}
}

func TestCheckToolsReferenceReal_NoSection(t *testing.T) {
	sf := &skillmd.SkillFile{Sections: map[string]*skillmd.Section{}}
	r := checkToolsReferenceReal(sf)
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q (no tools section)", r.Status, StatusPass)
	}
}

func TestCheckInstructionsSpecific_Valid(t *testing.T) {
	sf := loadFixture(t, "valid-agent-skill.md")
	r := checkInstructionsSpecific(sf)
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusPass, r.Message)
	}
}

func TestCheckInstructionsSpecific_Vague(t *testing.T) {
	sf := loadFixture(t, "vague-skill.md")
	r := checkInstructionsSpecific(sf)
	if r.Status != StatusWarn {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusWarn, r.Message)
	}
}

func TestCheckInstructionsSpecific_NoSection(t *testing.T) {
	sf := &skillmd.SkillFile{Sections: map[string]*skillmd.Section{}}
	r := checkInstructionsSpecific(sf)
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q (no instructions section)", r.Status, StatusPass)
	}
}

func TestCheckSkillFileNotTooLarge_WithinLimit(t *testing.T) {
	sf := loadFixture(t, "valid-agent-skill.md")
	r := checkSkillFileNotTooLarge(sf)
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusPass, r.Message)
	}
}

func TestCheckSkillFileNotTooLarge_ExceedsLimit(t *testing.T) {
	sf := loadFixture(t, "large-skill.md")
	r := checkSkillFileNotTooLarge(sf)
	if r.Status != StatusWarn {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusWarn, r.Message)
	}
}

func TestCheckSkillNameNotDuplicate_Unique(t *testing.T) {
	sf := loadFixture(t, "valid-agent-skill.md")
	allSkillNames := map[string]string{
		"other-skill": "/path/to/other/SKILL.md",
	}
	r := checkSkillNameNotDuplicate(sf, allSkillNames)
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusPass, r.Message)
	}
}

func TestCheckSkillNameNotDuplicate_Duplicate(t *testing.T) {
	sf := loadFixture(t, "valid-agent-skill.md")
	allSkillNames := map[string]string{
		"data-analyst": "/path/to/other/SKILL.md",
	}
	r := checkSkillNameNotDuplicate(sf, allSkillNames)
	if r.Status != StatusWarn {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusWarn, r.Message)
	}
}

func TestCheckSkillNameNotDuplicate_NoContext(t *testing.T) {
	sf := loadFixture(t, "valid-agent-skill.md")
	r := checkSkillNameNotDuplicate(sf, nil)
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q (skipped)", r.Status, StatusPass)
	}
}

// --- Integration tests for Validate with semantic checks ---

func TestValidate_ValidAgentSkill(t *testing.T) {
	dir := t.TempDir()
	data, err := readFile(testdataPath("valid-agent-skill.md"))
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}
	if err := writeFile(filepath.Join(dir, "SKILL.md"), data); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}

	result, err := Validate(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 20 checks now (15 original + 5 semantic)
	if result.Summary.Total != 20 {
		t.Errorf("total = %d, want 20", result.Summary.Total)
	}

	// Check that semantic checks passed
	semanticChecks := []string{
		CheckTriggersActionable,
		CheckToolsReferenceReal,
		CheckInstructionsSpecific,
		CheckSkillFileNotTooLarge,
		CheckSkillNameNotDuplicate,
	}
	for _, name := range semanticChecks {
		found := false
		for _, c := range result.Checks {
			if c.Name == name {
				found = true
				if c.Status != StatusPass {
					t.Errorf("check %q status = %q, want %q; message: %s", name, c.Status, StatusPass, c.Message)
				}
				break
			}
		}
		if !found {
			t.Errorf("check %q not found in results", name)
		}
	}
}

func TestValidate_VagueSkill(t *testing.T) {
	dir := t.TempDir()
	data, err := readFile(testdataPath("vague-skill.md"))
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}
	if err := writeFile(filepath.Join(dir, "SKILL.md"), data); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}

	result, err := Validate(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have warnings for vague triggers, tools, and instructions
	vagueChecks := map[string]bool{
		CheckTriggersActionable:   false,
		CheckToolsReferenceReal:   false,
		CheckInstructionsSpecific: false,
	}
	for _, c := range result.Checks {
		if _, ok := vagueChecks[c.Name]; ok {
			if c.Status == StatusWarn {
				vagueChecks[c.Name] = true
			}
		}
	}
	for name, warned := range vagueChecks {
		if !warned {
			t.Errorf("check %q did not warn for vague content", name)
		}
	}
}

func TestValidate_LargeSkill(t *testing.T) {
	dir := t.TempDir()
	data, err := readFile(testdataPath("large-skill.md"))
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}
	if err := writeFile(filepath.Join(dir, "SKILL.md"), data); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}

	result, err := Validate(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have warning for large file
	found := false
	for _, c := range result.Checks {
		if c.Name == CheckSkillFileNotTooLarge {
			found = true
			if c.Status != StatusWarn {
				t.Errorf("check %q status = %q, want %q; message: %s", c.Name, c.Status, StatusWarn, c.Message)
			}
			break
		}
	}
	if !found {
		t.Errorf("check %q not found in results", CheckSkillFileNotTooLarge)
	}
}
