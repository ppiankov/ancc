package validator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ppiankov/ancc/internal/skillmd"
)

// Check names.
const (
	CheckSkillMDExists          = "skill-md-exists"
	CheckSkillMDInstall         = "skill-md-install"
	CheckSkillMDCommands        = "skill-md-commands"
	CheckSkillMDFlags           = "skill-md-flags"
	CheckSkillMDJSON            = "skill-md-json-output"
	CheckSkillMDExitCodes       = "skill-md-exit-codes"
	CheckSkillMDNotDo           = "skill-md-not-do"
	CheckSkillMDParsing         = "skill-md-parsing"
	CheckHasInitCommand         = "has-init-command"
	CheckHasDoctorCommand       = "has-doctor-command"
	CheckHasBinaryRelease       = "has-binary-release"
	CheckJSONExamplesValid      = "json-examples-valid"
	CheckExitCodesNumeric       = "exit-codes-numeric"
	CheckCommandsNotPlaceholder = "commands-not-placeholder"
	CheckInstallHasCommand      = "install-has-command"
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

// skillMDPaths lists the locations where SKILL.md may be found, in priority order.
var skillMDPaths = []string{"SKILL.md", filepath.Join("docs", "SKILL.md")}

// findSkillMD returns the full path to SKILL.md within the repo, checking
// the root first and then docs/. Returns an empty string if not found.
func findSkillMD(repoPath string) string {
	for _, rel := range skillMDPaths {
		p := filepath.Join(repoPath, rel)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// checkSkillMDExists verifies SKILL.md exists at the repo root or docs/.
func checkSkillMDExists(path string) CheckResult {
	if found := findSkillMD(path); found != "" {
		return pass(CheckSkillMDExists, "SKILL.md found")
	}
	return fail(CheckSkillMDExists, "SKILL.md not found")
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

// --- Semantic quality checks ---

// checkJSONExamplesValid verifies all JSON code blocks in commands are parseable.
func checkJSONExamplesValid(sf *skillmd.SkillFile) CheckResult {
	checked := 0
	for _, cmd := range sf.Commands {
		if cmd.JSONOutput == "" {
			continue
		}
		checked++
		if !json.Valid([]byte(cmd.JSONOutput)) {
			return warn(CheckJSONExamplesValid,
				fmt.Sprintf("invalid JSON in %s output example", cmd.Name))
		}
	}
	if checked == 0 {
		return pass(CheckJSONExamplesValid, "no JSON examples to validate")
	}
	return pass(CheckJSONExamplesValid, fmt.Sprintf("%d JSON example(s) valid", checked))
}

// checkExitCodesNumeric verifies that at least one command defines exit code 0 (success).
func checkExitCodesNumeric(sf *skillmd.SkillFile) CheckResult {
	hasExitCodes := false
	hasZero := false
	for _, cmd := range sf.Commands {
		if len(cmd.ExitCodes) > 0 {
			hasExitCodes = true
			for _, ec := range cmd.ExitCodes {
				if ec.Code == 0 {
					hasZero = true
				}
			}
		}
	}
	if !hasExitCodes {
		return pass(CheckExitCodesNumeric, "no exit codes to validate")
	}
	if !hasZero {
		return warn(CheckExitCodesNumeric, "no command defines exit code 0 (success)")
	}
	return pass(CheckExitCodesNumeric, "exit code 0 (success) documented")
}

// placeholderNames are names that indicate unfilled template content.
var placeholderNames = []string{
	"mytool", "example", "placeholder", "yourapp", "yourtool", "myapp",
}

// checkCommandsNotPlaceholder warns if the tool name or commands use placeholder names.
func checkCommandsNotPlaceholder(sf *skillmd.SkillFile) CheckResult {
	name := strings.ToLower(sf.Name)
	for _, ph := range placeholderNames {
		if name == ph {
			return warn(CheckCommandsNotPlaceholder,
				fmt.Sprintf("tool name %q looks like a placeholder", sf.Name))
		}
	}
	for _, cmd := range sf.Commands {
		if strings.HasPrefix(cmd.Name, "<") {
			return warn(CheckCommandsNotPlaceholder,
				fmt.Sprintf("command %q looks like a template variable", cmd.Name))
		}
	}
	return pass(CheckCommandsNotPlaceholder, "no placeholder names detected")
}

// installCommands are prefixes that indicate a real install command.
var installCommands = []string{
	"brew ", "go install ", "npm ", "pip ", "cargo ", "apt ", "yum ",
	"dnf ", "curl ", "wget ", "docker ", "snap ",
}

// checkInstallHasCommand verifies the Install section contains a recognizable install command.
func checkInstallHasCommand(sf *skillmd.SkillFile) CheckResult {
	section := sf.Sections[skillmd.SectionInstall]
	if section == nil {
		return pass(CheckInstallHasCommand, "no Install section (checked elsewhere)")
	}

	inCodeBlock := false
	for _, line := range strings.Split(section.Content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}
		if !inCodeBlock {
			continue
		}
		for _, prefix := range installCommands {
			if strings.HasPrefix(trimmed, prefix) {
				return pass(CheckInstallHasCommand, "install command found")
			}
		}
	}
	return warn(CheckInstallHasCommand, "Install section has no recognizable install command")
}
