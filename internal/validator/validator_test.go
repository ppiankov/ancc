package validator

import (
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

	if result.Summary.Total != 11 {
		t.Errorf("total = %d, want 11", result.Summary.Total)
	}
	if result.Summary.Fail != 0 {
		t.Errorf("fail = %d, want 0", result.Summary.Fail)
	}
	// binary-release and doctor are warn for valid fixture with doctor
	// Actually valid-skill.md has doctor, so only binary-release warns.
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
	if result.Summary.Total != 11 {
		t.Errorf("total = %d, want 11", result.Summary.Total)
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
