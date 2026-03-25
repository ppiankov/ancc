package validator

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

	if result.Summary.Total != 33 {
		t.Errorf("total = %d, want 20", result.Summary.Total)
	}
	if result.Summary.Fail != 0 {
		t.Errorf("fail = %d, want 0", result.Summary.Fail)
	}
	// binary-release + changelog-exists + doctor-provenance warn for valid fixture in temp dir.
	if result.Summary.Warn != 3 {
		t.Errorf("warn = %d, want 3 (binary-release + changelog-exists + doctor-provenance)", result.Summary.Warn)
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
	if result.Summary.Total != 33 {
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

	if result.Summary.Total != 33 {
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

// --- New semantic quality check tests ---

func TestCheckTriggerActionable_NoTriggerSection(t *testing.T) {
	sf := &skillmd.SkillFile{
		Name: "testtool",
		Sections: map[string]*skillmd.Section{
			"Install": {Heading: "Install", Content: "brew install test"},
		},
	}
	r := checkTriggerActionable(sf)
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusPass, r.Message)
	}
}

func TestCheckTriggerActionable_ActionableTrigger(t *testing.T) {
	sf := &skillmd.SkillFile{
		Name: "testtool",
		Sections: map[string]*skillmd.Section{
			"When to use": {
				Heading: "When to use",
				Content: "Use this tool when you need to validate SKILL.md files.",
			},
		},
	}
	r := checkTriggerActionable(sf)
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusPass, r.Message)
	}
}

func TestCheckTriggerActionable_VagueTrigger(t *testing.T) {
	sf := &skillmd.SkillFile{
		Name: "testtool",
		Sections: map[string]*skillmd.Section{
			"When to use": {
				Heading: "When to use",
				Content: "Use this tool when needed for validation.",
			},
		},
	}
	r := checkTriggerActionable(sf)
	if r.Status != StatusWarn {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusWarn, r.Message)
	}
	if !strings.Contains(r.Message, "when needed") {
		t.Errorf("message = %q, want to contain %q", r.Message, "when needed")
	}
}

func TestCheckToolReferencesValid_NoToolSection(t *testing.T) {
	sf := &skillmd.SkillFile{
		Name: "testtool",
		Sections: map[string]*skillmd.Section{
			"Install": {Heading: "Install", Content: "brew install test"},
		},
	}
	r := checkToolReferencesValid(sf)
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusPass, r.Message)
	}
}

func TestCheckToolReferencesValid_KnownTools(t *testing.T) {
	sf := &skillmd.SkillFile{
		Name: "testtool",
		Sections: map[string]*skillmd.Section{
			"Tools": {
				Heading: "Tools",
				Content: "- `pastewatch` - scan for secrets\n- `git` - version control",
			},
		},
	}
	r := checkToolReferencesValid(sf)
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusPass, r.Message)
	}
}

func TestCheckToolReferencesValid_SuspiciousTools(t *testing.T) {
	sf := &skillmd.SkillFile{
		Name: "testtool",
		Sections: map[string]*skillmd.Section{
			"Tools": {
				Heading: "Tools",
				Content: "- `@@@invalid@@@` - suspicious tool\n- `xyz123!@#` - another suspicious tool",
			},
		},
	}
	r := checkToolReferencesValid(sf)
	if r.Status != StatusWarn {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusWarn, r.Message)
	}
}

func TestCheckInstructionsSpecific_NoVagueInstructions(t *testing.T) {
	sf := &skillmd.SkillFile{
		Name: "testtool",
		Sections: map[string]*skillmd.Section{
			"Usage": {
				Heading: "Usage",
				Content: "Run the tool with --format json to get structured output.",
			},
		},
	}
	r := checkInstructionsSpecific(sf)
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusPass, r.Message)
	}
}

func TestCheckInstructionsSpecific_VagueInstructions(t *testing.T) {
	sf := &skillmd.SkillFile{
		Name: "testtool",
		Sections: map[string]*skillmd.Section{
			"Usage": {
				Heading: "Usage",
				Content: "Handle errors appropriately as needed.",
			},
		},
	}
	r := checkInstructionsSpecific(sf)
	if r.Status != StatusWarn {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusWarn, r.Message)
	}
	if !strings.Contains(r.Message, "as needed") && !strings.Contains(r.Message, "handle appropriately") {
		t.Errorf("message = %q, want to contain vague phrase", r.Message)
	}
}

func TestCheckSkillLineCount_SmallFile(t *testing.T) {
	// Create a small temp file.
	dir := t.TempDir()
	path := filepath.Join(dir, "SKILL.md")
	content := "# test\n\nSmall file content.\n"
	if err := writeFile(path, []byte(content)); err != nil {
		t.Fatal(err)
	}
	r := checkSkillLineCount(path)
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusPass, r.Message)
	}
}

func TestCheckSkillLineCount_LargeFile(t *testing.T) {
	// Create a large temp file (>500 lines).
	dir := t.TempDir()
	path := filepath.Join(dir, "SKILL.md")
	var content strings.Builder
	content.WriteString("# test\n\n")
	for i := 0; i < 510; i++ {
		content.WriteString(fmt.Sprintf("Line %d of content\n", i))
	}
	if err := writeFile(path, []byte(content.String())); err != nil {
		t.Fatal(err)
	}
	r := checkSkillLineCount(path)
	if r.Status != StatusWarn {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusWarn, r.Message)
	}
	if !strings.Contains(r.Message, "lines (consider splitting)") {
		t.Errorf("message = %q, want to contain line count warning", r.Message)
	}
}

func TestCheckSkillLineCount_NoPath(t *testing.T) {
	r := checkSkillLineCount("")
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusPass, r.Message)
	}
}

func TestCheckDuplicateSkillNames_NoDuplicates(t *testing.T) {
	dir := t.TempDir()
	// Create a single SKILL.md.
	skillContent := "# testtool\n\nA test tool.\n"
	if err := writeFile(filepath.Join(dir, "SKILL.md"), []byte(skillContent)); err != nil {
		t.Fatal(err)
	}
	r := checkDuplicateSkillNames(dir)
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusPass, r.Message)
	}
}

func TestCheckDuplicateSkillNames_WithDuplicates(t *testing.T) {
	dir := t.TempDir()
	// Create two SKILL.md files with the same name.
	skillContent := "# testtool\n\nA test tool.\n"
	if err := writeFile(filepath.Join(dir, "SKILL.md"), []byte(skillContent)); err != nil {
		t.Fatal(err)
	}
	docsDir := filepath.Join(dir, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(filepath.Join(docsDir, "SKILL.md"), []byte(skillContent)); err != nil {
		t.Fatal(err)
	}
	r := checkDuplicateSkillNames(dir)
	if r.Status != StatusWarn {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusWarn, r.Message)
	}
	if !strings.Contains(r.Message, "testtool") {
		t.Errorf("message = %q, want to contain duplicate name", r.Message)
	}
}

func TestCheckDuplicateSkillNames_NoPath(t *testing.T) {
	r := checkDuplicateSkillNames("")
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusPass, r.Message)
	}
}

// --- Temporal contract check tests ---

func TestCheckChangelogExists_Present(t *testing.T) {
	dir := t.TempDir()
	if err := writeFile(filepath.Join(dir, "CHANGELOG.md"), []byte("# Changelog\n\n## [1.0.0]\n")); err != nil {
		t.Fatal(err)
	}
	r := checkChangelogExists(dir)
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusPass, r.Message)
	}
}

func TestCheckChangelogExists_Missing(t *testing.T) {
	r := checkChangelogExists(t.TempDir())
	if r.Status != StatusWarn {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusWarn, r.Message)
	}
}

func TestCheckChangelogVersionEntry_HasVersions(t *testing.T) {
	dir := t.TempDir()
	content := "# Changelog\n\n## [1.2.0] - 2026-03-25\n\n### Added\n\n- Feature\n\n## [1.1.0] - 2026-03-20\n\n### Fixed\n\n- Bug\n"
	if err := writeFile(filepath.Join(dir, "CHANGELOG.md"), []byte(content)); err != nil {
		t.Fatal(err)
	}
	r := checkChangelogVersionEntry(dir)
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusPass, r.Message)
	}
	if !strings.Contains(r.Message, "2 version(s)") {
		t.Errorf("message = %q, want to contain version count", r.Message)
	}
}

func TestCheckChangelogVersionEntry_NoVersionHeaders(t *testing.T) {
	dir := t.TempDir()
	content := "# Changelog\n\nSome text but no version headers.\n"
	if err := writeFile(filepath.Join(dir, "CHANGELOG.md"), []byte(content)); err != nil {
		t.Fatal(err)
	}
	r := checkChangelogVersionEntry(dir)
	if r.Status != StatusFail {
		t.Errorf("status = %q, want %q; message: %s", r.Status, StatusFail, r.Message)
	}
}

func TestCheckChangelogVersionEntry_NoChangelog(t *testing.T) {
	r := checkChangelogVersionEntry(t.TempDir())
	if r.Status != StatusPass {
		t.Errorf("status = %q, want %q (no CHANGELOG to check)", r.Status, StatusPass)
	}
}

func TestValidate_WithSemanticChecks(t *testing.T) {
	// Create a temp dir with a valid SKILL.md.
	dir := t.TempDir()
	data, err := readFile(testdataPath("valid-skill.md"))
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

	// Should have 30 checks now (15 original + 5 semantic + 5 scope + 5 spec).
	if result.Summary.Total != 33 {
		t.Errorf("total = %d, want 20", result.Summary.Total)
	}
}
