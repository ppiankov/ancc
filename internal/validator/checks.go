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
	// Semantic quality checks for agent SKILL.md files.
	CheckTriggersActionable     = "triggers-actionable"
	CheckToolsReferenceReal     = "tools-reference-real"
	CheckInstructionsSpecific   = "instructions-specific"
	CheckSkillFileNotTooLarge   = "skill-file-not-too-large"
	CheckSkillNameNotDuplicate  = "skill-name-not-duplicate"
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

// --- Semantic quality checks for agent SKILL.md files ---

// vagueTriggerPatterns are phrases that indicate non-actionable triggers.
var vagueTriggerPatterns = []string{
	"when needed",
	"when appropriate",
	"as needed",
	"if necessary",
	"when relevant",
	"when suitable",
	"at discretion",
	"when required",
	"when useful",
}

// checkTriggersActionable verifies trigger sections contain actionable conditions.
// Warns if triggers use vague language like "when needed" or "as appropriate".
func checkTriggersActionable(sf *skillmd.SkillFile) CheckResult {
	// Look for trigger-related sections.
	triggerSections := []string{"When to Use", "Triggers", "When to Use This Skill", "Usage Triggers"}
	var content string
	for _, name := range triggerSections {
		if section, ok := sf.Sections[name]; ok {
			content = section.Content
			break
		}
	}
	if content == "" {
		return pass(CheckTriggersActionable, "no trigger section found (may not apply)")
	}

	lower := strings.ToLower(content)
	for _, pattern := range vagueTriggerPatterns {
		if strings.Contains(lower, pattern) {
			return warn(CheckTriggersActionable,
				fmt.Sprintf("trigger section contains vague condition %q", pattern))
		}
	}
	return pass(CheckTriggersActionable, "trigger conditions appear actionable")
}

// knownMCPPrefixes are recognized MCP tool prefixes.
var knownMCPPrefixes = []string{
	"pastewatch", "contextspectre", "tokencontrol", "workledger",
	"notion", "github", "slack", "linear", "jira", "confluence",
	"postgres", "mysql", "redis", "mongodb", "sqlite",
	"filesystem", "puppeteer", "playwright", "selenium",
	"docker", "kubernetes", "kubectl", "helm",
	"aws", "gcp", "azure", "cloudflare",
	"stripe", "sendgrid", "twilio", "slack",
	"git", "npm", "pip", "cargo", "go",
}

// suspiciousToolPatterns indicate potentially fake tool names.
var suspiciousToolPatterns = []string{
	"my-mcp", "example-mcp", "placeholder", "your-mcp",
	"<mcp", "${mcp", "{{mcp", "todo", "FIXME", "XXX",
}

// checkToolsReferenceReal verifies tool lists reference real MCP tools or CLI commands.
// Warns on suspicious names that look like placeholders.
func checkToolsReferenceReal(sf *skillmd.SkillFile) CheckResult {
	// Look for tools-related sections.
	toolSections := []string{"Tools", "MCP Tools", "MCP Servers", "Commands", "Integrations"}
	var content string
	for _, name := range toolSections {
		if section, ok := sf.Sections[name]; ok {
			content = section.Content
			break
		}
	}
	if content == "" {
		return pass(CheckToolsReferenceReal, "no tools section found (may not apply)")
	}

	lower := strings.ToLower(content)
	for _, pattern := range suspiciousToolPatterns {
		if strings.Contains(lower, strings.ToLower(pattern)) {
			return warn(CheckToolsReferenceReal,
				fmt.Sprintf("tools section contains suspicious name %q", pattern))
		}
	}
	return pass(CheckToolsReferenceReal, "tool names appear valid")
}

// vagueInstructionPatterns are phrases that indicate non-specific instructions.
var vagueInstructionPatterns = []string{
	"handle appropriately",
	"deal with it",
	"manage accordingly",
	"as appropriate",
	"use your judgment",
	"decide based on context",
	"handle as needed",
	"take care of",
	"sort it out",
	"figure it out",
	"do what's best",
	"apply best practices",
	"follow conventions",
}

// checkInstructionsSpecific verifies instructions are specific enough.
// Warns on overly vague instructions like "handle appropriately".
func checkInstructionsSpecific(sf *skillmd.SkillFile) CheckResult {
	// Look for instruction-related sections.
	instructionSections := []string{"Instructions", "How to Use", "Usage", "Procedure", "Steps"}
	var content string
	for _, name := range instructionSections {
		if section, ok := sf.Sections[name]; ok {
			content = section.Content
			break
		}
	}
	if content == "" {
		return pass(CheckInstructionsSpecific, "no instructions section found (may not apply)")
	}

	lower := strings.ToLower(content)
	for _, pattern := range vagueInstructionPatterns {
		if strings.Contains(lower, pattern) {
			return warn(CheckInstructionsSpecific,
				fmt.Sprintf("instructions contain vague phrase %q", pattern))
		}
	}
	return pass(CheckInstructionsSpecific, "instructions appear specific")
}

// checkSkillFileNotTooLarge warns if a skill file exceeds 500 lines.
// Large skill files are likely too complex and should be split.
func checkSkillFileNotTooLarge(sf *skillmd.SkillFile) CheckResult {
	if sf.LineCount > 500 {
		return warn(CheckSkillFileNotTooLarge,
			fmt.Sprintf("skill file has %d lines (consider splitting if >500)", sf.LineCount))
	}
	return pass(CheckSkillFileNotTooLarge, fmt.Sprintf("skill file size (%d lines) is reasonable", sf.LineCount))
}

// checkSkillNameNotDuplicate checks for duplicate skill names.
// This check requires external context (other skill files) and is a placeholder
// for batch validation scenarios. For single-file validation, it always passes.
func checkSkillNameNotDuplicate(sf *skillmd.SkillFile, allSkillNames map[string]string) CheckResult {
	if allSkillNames == nil {
		return pass(CheckSkillNameNotDuplicate, "duplicate check skipped (no context)")
	}
	if existingPath, ok := allSkillNames[sf.Name]; ok {
		return warn(CheckSkillNameNotDuplicate,
			fmt.Sprintf("skill name %q duplicates %s", sf.Name, existingPath))
	}
	return pass(CheckSkillNameNotDuplicate, fmt.Sprintf("skill name %q is unique", sf.Name))
}
