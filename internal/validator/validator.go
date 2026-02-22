package validator

import (
	"fmt"
	"path/filepath"

	"github.com/ppiankov/ancc/internal/skillmd"
)

// Validate runs all checks against the repo at path and returns results.
func Validate(path string) (*ValidationResult, error) {
	// Check if path is a GitHub URL.
	if gh := ParseGitHubURL(path); gh != nil {
		return ValidateGitHub(gh.Owner, gh.Repo)
	}

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
			checkBinaryRelease(""),
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
		checkBinaryRelease(""),
	)

	computeSummary(result)
	return result, nil
}

// ValidateGitHub runs all checks against a GitHub repo.
func ValidateGitHub(owner, repo string) (*ValidationResult, error) {
	client := newGitHubClient()
	return validateGitHubWithClient(client, owner, repo)
}

// validateGitHubWithClient is the testable core of ValidateGitHub.
func validateGitHubWithClient(client *gitHubClient, owner, repo string) (*ValidationResult, error) {
	ref := fmt.Sprintf("github.com/%s/%s", owner, repo)
	result := &ValidationResult{Path: ref}

	// Fetch SKILL.md.
	content, err := client.FetchSkillMD(owner, repo)
	if err != nil {
		// SKILL.md not found — fail all content checks.
		result.Checks = append(result.Checks,
			fail(CheckSkillMDExists, fmt.Sprintf("SKILL.md not found in %s/%s", owner, repo)),
			fail(CheckSkillMDInstall, "SKILL.md not found"),
			fail(CheckSkillMDCommands, "SKILL.md not found"),
			fail(CheckSkillMDFlags, "SKILL.md not found"),
			fail(CheckSkillMDJSON, "SKILL.md not found"),
			fail(CheckSkillMDExitCodes, "SKILL.md not found"),
			fail(CheckSkillMDNotDo, "SKILL.md not found"),
			fail(CheckSkillMDParsing, "SKILL.md not found"),
			fail(CheckHasInitCommand, "SKILL.md not found"),
			warn(CheckHasDoctorCommand, "SKILL.md not found"),
		)
		// Still check binary releases.
		result.Checks = append(result.Checks, checkBinaryReleaseGitHub(client, owner, repo))
		computeSummary(result)
		return result, nil
	}

	result.Checks = append(result.Checks, pass(CheckSkillMDExists, "SKILL.md found in GitHub repo"))

	// Parse content.
	sf, err := skillmd.Parse(content)
	if err != nil {
		return nil, fmt.Errorf("parsing SKILL.md: %w", err)
	}

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
		checkBinaryReleaseGitHub(client, owner, repo),
	)

	computeSummary(result)
	return result, nil
}

// checkBinaryReleaseGitHub checks GitHub releases for binary assets.
func checkBinaryReleaseGitHub(client *gitHubClient, owner, repo string) CheckResult {
	hasBinary, err := client.HasBinaryRelease(owner, repo)
	if err != nil {
		return warn(CheckHasBinaryRelease, fmt.Sprintf("could not check releases: %v", err))
	}
	if !hasBinary {
		return warn(CheckHasBinaryRelease, "no binary release assets found")
	}
	return pass(CheckHasBinaryRelease, "binary release assets found")
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
