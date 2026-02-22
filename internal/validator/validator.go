package validator

import (
	"fmt"
	"path/filepath"

	"github.com/ppiankov/ancc/internal/skillmd"
)

// Validate runs all checks against the repo at path and returns results.
func Validate(path string) (*ValidationResult, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	result := &ValidationResult{Path: path}

	// Check 1: SKILL.md exists (filesystem check).
	existsResult := checkSkillMDExists(path)
	result.Checks = append(result.Checks, existsResult)

	// If SKILL.md doesn't exist, remaining checks fail.
	if existsResult.Status == StatusFail {
		result.Checks = append(result.Checks,
			fail(CheckSkillMDInstall, "SKILL.md not found"),
			fail(CheckSkillMDCommands, "SKILL.md not found"),
			fail(CheckSkillMDFlags, "SKILL.md not found"),
			fail(CheckSkillMDJSON, "SKILL.md not found"),
			fail(CheckSkillMDExitCodes, "SKILL.md not found"),
			fail(CheckSkillMDNotDo, "SKILL.md not found"),
			fail(CheckSkillMDParsing, "SKILL.md not found"),
			fail(CheckHasInitCommand, "SKILL.md not found"),
			warn(CheckHasDoctorCommand, "SKILL.md not found"),
			checkBinaryRelease(path),
		)
		computeSummary(result)
		return result, nil
	}

	// Parse SKILL.md.
	sf, err := skillmd.ParseFile(filepath.Join(path, "SKILL.md"))
	if err != nil {
		return nil, fmt.Errorf("parsing SKILL.md: %w", err)
	}

	// Run content checks.
	result.Checks = append(result.Checks,
		checkInstall(sf),
		checkCommands(sf),
		checkFlags(sf),
		checkJSONOutput(sf),
		checkExitCodes(sf),
		checkNotDo(sf),
		checkParsing(sf),
		checkInitCommand(sf),
		checkDoctorCommand(sf),
		checkBinaryRelease(path),
	)

	computeSummary(result)
	return result, nil
}

// computeSummary tallies results and sets the overall status.
func computeSummary(r *ValidationResult) {
	for _, c := range r.Checks {
		r.Summary.Total++
		switch c.Status {
		case StatusPass:
			r.Summary.Pass++
		case StatusFail:
			r.Summary.Fail++
		case StatusWarn:
			r.Summary.Warn++
		}
	}

	switch {
	case r.Summary.Fail > 0:
		r.Status = OverallFail
	case r.Summary.Warn > 0:
		r.Status = OverallPartial
	default:
		r.Status = OverallPass
	}
}
