package validator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ppiankov/ancc/internal/skillmd"
)

// Check names.
const (
	CheckSkillMDExists    = "skill-md-exists"
	CheckSkillMDInstall   = "skill-md-install"
	CheckSkillMDCommands  = "skill-md-commands"
	CheckSkillMDFlags     = "skill-md-flags"
	CheckSkillMDJSON      = "skill-md-json-output"
	CheckSkillMDExitCodes = "skill-md-exit-codes"
	CheckSkillMDNotDo     = "skill-md-not-do"
	CheckSkillMDParsing   = "skill-md-parsing"
	CheckHasInitCommand   = "has-init-command"
	CheckHasDoctorCommand = "has-doctor-command"
	CheckHasBinaryRelease = "has-binary-release"
)

func pass(name, msg string) CheckResult {
	return CheckResult{Name: name, Status: StatusPass, Message: msg}
}

func fail(name, msg string) CheckResult {
	return CheckResult{Name: name, Status: StatusFail, Message: msg}
}

func warn(name, msg string) CheckResult {
	return CheckResult{Name: name, Status: StatusWarn, Message: msg}
}

// checkSkillMDExists verifies SKILL.md exists at the repo root.
func checkSkillMDExists(path string) CheckResult {
	p := filepath.Join(path, "SKILL.md")
	if _, err := os.Stat(p); err != nil {
		return fail(CheckSkillMDExists, "SKILL.md not found at repo root")
	}
	return pass(CheckSkillMDExists, "SKILL.md found at repo root")
}

// checkInstall verifies the Install section exists.
func checkInstall(sf *skillmd.SkillFile) CheckResult {
	if sf.Sections[skillmd.SectionInstall] == nil {
		return fail(CheckSkillMDInstall, "missing ## Install section")
	}
	return pass(CheckSkillMDInstall, "Install section found")
}

// checkCommands verifies the Commands section exists with at least one command.
func checkCommands(sf *skillmd.SkillFile) CheckResult {
	if sf.Sections[skillmd.SectionCommands] == nil {
		return fail(CheckSkillMDCommands, "missing ## Commands section")
	}
	if len(sf.Commands) == 0 {
		return fail(CheckSkillMDCommands, "Commands section has no documented commands")
	}
	return pass(CheckSkillMDCommands, fmt.Sprintf("%d command(s) documented", len(sf.Commands)))
}

// checkFlags verifies at least one command documents --format json.
func checkFlags(sf *skillmd.SkillFile) CheckResult {
	for _, cmd := range sf.Commands {
		for _, f := range cmd.Flags {
			if strings.Contains(f.Name, "--format json") {
				return pass(CheckSkillMDFlags, "--format json flag documented")
			}
		}
	}
	return fail(CheckSkillMDFlags, "no command documents --format json flag")
}

// checkJSONOutput verifies at least one command shows a JSON output schema.
func checkJSONOutput(sf *skillmd.SkillFile) CheckResult {
	for _, cmd := range sf.Commands {
		if cmd.JSONOutput != "" {
			return pass(CheckSkillMDJSON, "JSON output schema documented")
		}
	}
	return fail(CheckSkillMDJSON, "no command shows JSON output schema")
}

// checkExitCodes verifies at least one command documents exit codes.
func checkExitCodes(sf *skillmd.SkillFile) CheckResult {
	for _, cmd := range sf.Commands {
		if len(cmd.ExitCodes) > 0 {
			return pass(CheckSkillMDExitCodes, "exit codes documented")
		}
	}
	return fail(CheckSkillMDExitCodes, "no command documents exit codes")
}

// checkNotDo verifies the "What this does NOT do" section exists.
func checkNotDo(sf *skillmd.SkillFile) CheckResult {
	if sf.Sections[skillmd.SectionWhatNotDo] == nil {
		return fail(CheckSkillMDNotDo, "missing \"What this does NOT do\" section")
	}
	return pass(CheckSkillMDNotDo, "\"What this does NOT do\" section found")
}

// checkParsing verifies the parsing examples section exists.
func checkParsing(sf *skillmd.SkillFile) CheckResult {
	if sf.Sections[skillmd.SectionParsingExamples] == nil {
		return fail(CheckSkillMDParsing, "missing \"Parsing examples\" section")
	}
	return pass(CheckSkillMDParsing, "Parsing examples section found")
}

// checkInitCommand verifies a command named "init" is documented.
func checkInitCommand(sf *skillmd.SkillFile) CheckResult {
	for _, cmd := range sf.Commands {
		if strings.HasSuffix(cmd.Name, " init") || cmd.Name == "init" {
			return pass(CheckHasInitCommand, "init command documented")
		}
	}
	return fail(CheckHasInitCommand, "no init command documented")
}

// checkDoctorCommand verifies a command named "doctor" is documented.
// This is a recommended extension — warn, not fail.
func checkDoctorCommand(sf *skillmd.SkillFile) CheckResult {
	for _, cmd := range sf.Commands {
		if strings.HasSuffix(cmd.Name, " doctor") || cmd.Name == "doctor" {
			return pass(CheckHasDoctorCommand, "doctor command documented")
		}
	}
	return warn(CheckHasDoctorCommand, "no doctor command documented (recommended)")
}

// checkBinaryRelease checks for binary release assets.
// For now, this only applies to GitHub repos and warns otherwise.
func checkBinaryRelease(_ string) CheckResult {
	// GitHub release checking is WO-005 scope.
	return warn(CheckHasBinaryRelease, "binary release check requires GitHub URL (skipped)")
}
