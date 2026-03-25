package validator

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
	// Not-do quality checks.
	CheckNotDoMinItems      = "not-do-min-items"
	CheckNotDoSpecificity   = "not-do-specificity"
	CheckNotDoNoOverlap     = "not-do-no-overlap"
	CheckNotDoBoundaryVerbs = "not-do-boundary-verbs"
	// Scope pressure check.
	CheckScopePressure = "scope-pressure"
	// Phase 2: spec + policy checks.
	CheckDoctorOutputValid    = "doctor-output-valid"
	CheckHandoffSection       = "handoff-section"
	CheckProvenanceDocumented = "provenance-documented"
	CheckDeprecatedCommands   = "deprecated-commands"
	CheckFailureModes         = "failure-modes-documented"
	// Semantic quality checks.
	CheckTriggerActionable    = "trigger-actionable"
	CheckToolReferencesValid  = "tool-references-valid"
	CheckInstructionsSpecific = "instructions-specific"
	CheckSkillLineCount       = "skill-line-count"
	CheckDuplicateSkillNames  = "duplicate-skill-names"
	// Temporal contract checks.
	CheckChangelogExists       = "changelog-exists"
	CheckChangelogVersionEntry = "changelog-version-entry"
	// Doctor provenance check.
	CheckDoctorProvenance = "doctor-provenance"
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

// --- Semantic quality checks for SKILL.md ---

// vaguePhrases are phrases that indicate overly vague instructions.
var vaguePhrases = []string{
	"handle appropriately",
	"as needed",
	"when needed",
	"as appropriate",
	"if necessary",
	"if needed",
	"as required",
	"properly",
	"correctly",
	"appropriately",
	"relevant",
	"suitable",
	"adequate",
}

// actionablePatterns are patterns that indicate actionable trigger conditions.
var actionablePatterns = []string{
	"when ",
	"if ",
	"before ",
	"after ",
	"during ",
	"on ",
	"upon ",
	"whenever ",
	"each time ",
	"every time ",
}

// checkTriggerActionable verifies that trigger sections contain actionable conditions.
// Warns if triggers use vague language like "when needed" instead of specific conditions.
func checkTriggerActionable(sf *skillmd.SkillFile) CheckResult {
	// Look for sections that might contain triggers (e.g., "When to use", "Triggers", "Usage").
	triggerHeadings := []string{"when", "trigger", "usage", "when to use", "triggers"}

	var foundTrigger bool
	var hasVagueTrigger bool
	var vagueExample string

	for heading, section := range sf.Sections {
		headingLower := strings.ToLower(heading)
		isTriggerSection := false
		for _, th := range triggerHeadings {
			if strings.Contains(headingLower, th) {
				isTriggerSection = true
				break
			}
		}
		if !isTriggerSection {
			continue
		}

		foundTrigger = true
		contentLower := strings.ToLower(section.Content)

		// Check if the section contains vague phrases.
		for _, phrase := range vaguePhrases {
			if strings.Contains(contentLower, phrase) {
				hasVagueTrigger = true
				vagueExample = phrase
				break
			}
		}

		// Check if the section contains actionable patterns.
		hasActionable := false
		for _, pattern := range actionablePatterns {
			if strings.Contains(contentLower, pattern) {
				hasActionable = true
				break
			}
		}

		// If no actionable pattern and has vague phrase, warn.
		if !hasActionable && hasVagueTrigger {
			break
		}
	}

	if !foundTrigger {
		return pass(CheckTriggerActionable, "no trigger section found (optional)")
	}
	if hasVagueTrigger {
		return warn(CheckTriggerActionable, fmt.Sprintf("trigger section contains vague phrase %q", vagueExample))
	}
	return pass(CheckTriggerActionable, "trigger conditions are actionable")
}

// knownMCPServers is a list of known MCP server names for validation.
var knownMCPServers = []string{
	"pastewatch", "contextspectre", "tokencontrol", "workledger",
	"filesystem", "git", "github", "memory", "time", "puppeteer",
	"playwright", "postgres", "sqlite", "redis", "notion", "slack",
	"google-maps", "google-drive", "dropbox", "figma", "linear",
	"jira", "confluence", "stripe", "sendgrid", "twilio",
}

// knownCLICommands is a list of known CLI command prefixes.
var knownCLICommands = []string{
	"brew ", "go ", "npm ", "pip ", "cargo ", "apt ", "yum ",
	"dnf ", "curl ", "wget ", "docker ", "snap ", "git ", "gh ",
	"aws ", "gcloud ", "az ", "kubectl ", "helm ", "terraform ",
	"ansible ", "node ", "python ", "ruby ", "java ", "mvn ",
	"gradle ", "make ", "cmake ", "gcc ", "clang ", "rustc ",
}

// checkToolReferencesValid verifies that tool lists reference real MCP tools or CLI commands.
// Warns on suspicious names that don't match known patterns.
func checkToolReferencesValid(sf *skillmd.SkillFile) CheckResult {
	// Look for sections that might contain tool references.
	toolHeadings := []string{"tools", "mcp", "servers", "integrations", "dependencies", "requirements"}

	var foundTools bool
	var suspiciousTools []string

	for heading, section := range sf.Sections {
		headingLower := strings.ToLower(heading)
		isToolSection := false
		for _, th := range toolHeadings {
			if strings.Contains(headingLower, th) {
				isToolSection = true
				break
			}
		}
		if !isToolSection {
			continue
		}

		foundTools = true

		// Extract potential tool names from the section content.
		lines := strings.Split(section.Content, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			// Look for list items or code references.
			if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") ||
				strings.HasPrefix(trimmed, "`") || strings.HasPrefix(trimmed, "- `") {
				// Extract potential tool name.
				name := strings.TrimPrefix(trimmed, "- ")
				name = strings.TrimPrefix(name, "* ")
				name = strings.TrimPrefix(name, "`")
				name = strings.TrimSuffix(name, "`")
				name = strings.TrimSpace(name)
				// Remove leading dash if present (from "- `tool`" pattern).
				name = strings.TrimPrefix(name, "- `")
				name = strings.TrimSuffix(name, "`")
				name = strings.TrimSpace(name)

				if name == "" || len(name) < 2 {
					continue
				}

				// Check if it matches known patterns.
				isKnown := false
				nameLower := strings.ToLower(name)

				// Check against known MCP servers.
				for _, known := range knownMCPServers {
					if strings.Contains(nameLower, known) || strings.Contains(known, nameLower) {
						isKnown = true
						break
					}
				}

				// Check against known CLI commands.
				if !isKnown {
					for _, known := range knownCLICommands {
						if strings.HasPrefix(nameLower, strings.TrimSpace(known)) {
							isKnown = true
							break
						}
					}
				}

				// Check if it looks like a valid tool name (alphanumeric with dashes/underscores).
				if !isKnown {
					validName := true
					for _, r := range name {
						if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
							(r >= '0' && r <= '9') || r == '-' || r == '_' || r == '/') {
							validName = false
							break
						}
					}
					if validName && len(name) > 2 {
						// Looks like a valid name, just not in our known list.
						isKnown = true
					}
				}

				if !isKnown {
					suspiciousTools = append(suspiciousTools, name)
				}
			}
		}
	}

	if !foundTools {
		return pass(CheckToolReferencesValid, "no tool section found (optional)")
	}
	if len(suspiciousTools) > 0 {
		return warn(CheckToolReferencesValid, fmt.Sprintf("suspicious tool names: %v", suspiciousTools))
	}
	return pass(CheckToolReferencesValid, "all tool references look valid")
}

// checkInstructionsSpecific verifies that instructions are specific and not overly vague.
// Warns on phrases like "handle appropriately" that don't provide actionable guidance.
func checkInstructionsSpecific(sf *skillmd.SkillFile) CheckResult {
	var vagueLocations []string

	for heading, section := range sf.Sections {
		contentLower := strings.ToLower(section.Content)
		for _, phrase := range vaguePhrases {
			if strings.Contains(contentLower, phrase) {
				vagueLocations = append(vagueLocations, fmt.Sprintf("%s: %q", heading, phrase))
				break
			}
		}
	}

	if len(vagueLocations) == 0 {
		return pass(CheckInstructionsSpecific, "no vague instructions found")
	}
	return warn(CheckInstructionsSpecific, fmt.Sprintf("vague instructions: %v", vagueLocations))
}

// checkSkillLineCount warns if a skill file exceeds 500 lines.
// Large skill files may indicate overly complex configurations.
func checkSkillLineCount(path string) CheckResult {
	if path == "" {
		return pass(CheckSkillLineCount, "line count check requires file path")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return warn(CheckSkillLineCount, fmt.Sprintf("could not read file: %v", err))
	}

	lines := strings.Count(string(data), "\n") + 1
	if lines > 500 {
		return warn(CheckSkillLineCount, fmt.Sprintf("skill file has %d lines (consider splitting)", lines))
	}
	return pass(CheckSkillLineCount, fmt.Sprintf("skill file has %d lines", lines))
}

// --- Not-do quality checks ---

// notDoBullets extracts bullet items from the not-do section.
func notDoBullets(sf *skillmd.SkillFile) []string {
	section := sf.Sections[skillmd.SectionWhatNotDo]
	if section == nil {
		return nil
	}
	var bullets []string
	for _, line := range strings.Split(section.Content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- ") {
			bullets = append(bullets, trimmed)
		}
	}
	return bullets
}

// checkNotDoMinItems verifies the not-do section has at least 3 bullet points.
func checkNotDoMinItems(sf *skillmd.SkillFile) CheckResult {
	if sf.Sections[skillmd.SectionWhatNotDo] == nil {
		return pass(CheckNotDoMinItems, "no not-do section (checked elsewhere)")
	}
	bullets := notDoBullets(sf)
	if len(bullets) < 3 {
		return fail(CheckNotDoMinItems, fmt.Sprintf("not-do section has %d items, need at least 3", len(bullets)))
	}
	return pass(CheckNotDoMinItems, fmt.Sprintf("not-do section has %d items", len(bullets)))
}

// vagueNotDoPhrases are patterns indicating vague not-do items.
var vagueNotDoPhrases = []string{
	"anything specific",
	"everything else",
	"other things",
	"whatever",
	"bad things",
	"harm the system",
	"not responsible",
}

// checkNotDoSpecificity warns on vague not-do items.
func checkNotDoSpecificity(sf *skillmd.SkillFile) CheckResult {
	bullets := notDoBullets(sf)
	if len(bullets) == 0 {
		return pass(CheckNotDoSpecificity, "no not-do items to check")
	}
	for _, bullet := range bullets {
		lower := strings.ToLower(bullet)
		for _, phrase := range vagueNotDoPhrases {
			if strings.Contains(lower, phrase) {
				return warn(CheckNotDoSpecificity, fmt.Sprintf("vague not-do item: %s", bullet))
			}
		}
	}
	return pass(CheckNotDoSpecificity, "not-do items are specific")
}

// checkNotDoNoOverlap warns if not-do items directly contradict documented commands.
// Only flags when a not-do item says "does not <subcmd>" as a standalone verb,
// not when the subcmd word appears as part of a longer phrase.
func checkNotDoNoOverlap(sf *skillmd.SkillFile) CheckResult {
	bullets := notDoBullets(sf)
	if len(bullets) == 0 {
		return pass(CheckNotDoNoOverlap, "no not-do items to check")
	}
	for _, cmd := range sf.Commands {
		parts := strings.Fields(cmd.Name)
		if len(parts) == 0 {
			continue
		}
		subCmd := strings.ToLower(parts[len(parts)-1])
		if len(subCmd) < 3 {
			continue
		}
		for _, bullet := range bullets {
			lower := strings.ToLower(bullet)
			// Check for "does not <subcmd>" followed by end-of-string or non-alpha.
			// This avoids false positives like "does not validate code quality"
			// matching command "validate" — only flags direct contradictions like
			// "does not deploy" when "deploy" is a command.
			for _, prefix := range []string{"does not " + subCmd, "not " + subCmd} {
				idx := strings.Index(lower, prefix)
				if idx < 0 {
					continue
				}
				afterIdx := idx + len(prefix)
				if afterIdx >= len(lower) {
					// Exact match at end of string — direct contradiction.
					return warn(CheckNotDoNoOverlap,
						fmt.Sprintf("not-do item contradicts command %q: %s", cmd.Name, bullet))
				}
				// Only flag if subcmd is at end of meaningful phrase (punctuation or EOL).
				// "does not deploy" or "does not deploy." = contradiction.
				// "does not validate code quality" = subcmd used as verb, not contradiction.
				next := lower[afterIdx]
				if next == ',' || next == '.' || next == ';' || next == ':' || next == '\n' {
					return warn(CheckNotDoNoOverlap,
						fmt.Sprintf("not-do item contradicts command %q: %s", cmd.Name, bullet))
				}
			}
		}
	}
	return pass(CheckNotDoNoOverlap, "no overlap between not-do items and commands")
}

// boundaryVerbs are verbs that define clear scope boundaries.
var boundaryVerbs = []string{
	"manage", "store", "execute", "replace", "modify", "own",
	"install", "deploy", "access", "control", "authenticate",
	"persist", "cache", "monitor", "orchestrate",
}

// checkNotDoBoundaryVerbs warns if not-do items lack boundary-defining verbs.
func checkNotDoBoundaryVerbs(sf *skillmd.SkillFile) CheckResult {
	bullets := notDoBullets(sf)
	if len(bullets) == 0 {
		return pass(CheckNotDoBoundaryVerbs, "no not-do items to check")
	}
	for _, bullet := range bullets {
		lower := strings.ToLower(bullet)
		for _, verb := range boundaryVerbs {
			if strings.Contains(lower, verb) {
				return pass(CheckNotDoBoundaryVerbs, "not-do items use boundary-defining verbs")
			}
		}
	}
	return warn(CheckNotDoBoundaryVerbs, "not-do items lack boundary verbs (manage/store/execute/replace/modify/own)")
}

// checkScopePressure warns when a SKILL.md has many commands or sections.
func checkScopePressure(sf *skillmd.SkillFile) CheckResult {
	if len(sf.Commands) > 10 {
		return warn(CheckScopePressure,
			fmt.Sprintf("tool has %d commands; consider splitting into focused tools", len(sf.Commands)))
	}
	if len(sf.Sections) > 8 {
		return warn(CheckScopePressure,
			fmt.Sprintf("skill file has %d sections; consider splitting", len(sf.Sections)))
	}
	return pass(CheckScopePressure, fmt.Sprintf("%d commands, %d sections", len(sf.Commands), len(sf.Sections)))
}

// --- Phase 2 checks ---

// checkDoctorOutputValid validates doctor command JSON output has required fields.
func checkDoctorOutputValid(sf *skillmd.SkillFile) CheckResult {
	for _, cmd := range sf.Commands {
		if !strings.HasSuffix(cmd.Name, " doctor") && cmd.Name != "doctor" {
			continue
		}
		if cmd.JSONOutput == "" {
			return pass(CheckDoctorOutputValid, "doctor command has no JSON example (optional)")
		}
		var doc map[string]interface{}
		if err := json.Unmarshal([]byte(cmd.JSONOutput), &doc); err != nil {
			return warn(CheckDoctorOutputValid, "doctor JSON output is not valid JSON")
		}
		if _, ok := doc["status"]; !ok {
			return warn(CheckDoctorOutputValid, "doctor output missing required 'status' field")
		}
		checks, ok := doc["checks"]
		if !ok {
			return warn(CheckDoctorOutputValid, "doctor output missing required 'checks' array")
		}
		if arr, ok := checks.([]interface{}); ok && len(arr) > 0 {
			if entry, ok := arr[0].(map[string]interface{}); ok {
				if _, ok := entry["name"]; !ok {
					return warn(CheckDoctorOutputValid, "doctor checks[0] missing 'name' field")
				}
				if _, ok := entry["status"]; !ok {
					return warn(CheckDoctorOutputValid, "doctor checks[0] missing 'status' field")
				}
			}
		}
		return pass(CheckDoctorOutputValid, "doctor JSON output matches schema")
	}
	return pass(CheckDoctorOutputValid, "no doctor command (checked elsewhere)")
}

// checkHandoffSection recommends a Handoffs section when the tool has 3+ commands.
func checkHandoffSection(sf *skillmd.SkillFile) CheckResult {
	if sf.Sections[skillmd.SectionHandoffs] != nil {
		return pass(CheckHandoffSection, "Handoffs section found")
	}
	if len(sf.Commands) >= 3 {
		return warn(CheckHandoffSection, "3+ commands documented; consider adding ## Handoffs section")
	}
	return pass(CheckHandoffSection, "Handoffs section not needed (<3 commands)")
}

// checkProvenanceDocumented warns if JSON examples mix provenance signals without a provenance field.
func checkProvenanceDocumented(sf *skillmd.SkillFile) CheckResult {
	provenanceValues := []string{"observed", "declared", "inferred", "unknown"}
	for _, cmd := range sf.Commands {
		if cmd.JSONOutput == "" {
			continue
		}
		lower := strings.ToLower(cmd.JSONOutput)
		hasProvenanceField := strings.Contains(lower, "\"provenance\"") || strings.Contains(lower, "\"source\"")
		mixedSignals := 0
		for _, pv := range provenanceValues {
			if strings.Contains(lower, "\""+pv+"\"") {
				mixedSignals++
			}
		}
		if mixedSignals >= 2 && !hasProvenanceField {
			return warn(CheckProvenanceDocumented, fmt.Sprintf("command %s mixes provenance values without a provenance field", cmd.Name))
		}
	}
	return pass(CheckProvenanceDocumented, "no provenance issues detected")
}

// deprecatedKeywords are words indicating a command is deprecated.
var deprecatedKeywords = []string{"deprecated", "legacy", "obsolete", "removed", "do not use"}

// checkDeprecatedCommands warns if commands appear deprecated but no Deprecated section exists.
func checkDeprecatedCommands(sf *skillmd.SkillFile) CheckResult {
	if sf.Sections[skillmd.SectionDeprecated] != nil {
		return pass(CheckDeprecatedCommands, "Deprecated section documents lifecycle")
	}
	for _, cmd := range sf.Commands {
		lower := strings.ToLower(cmd.Name + " " + cmd.Desc)
		for _, kw := range deprecatedKeywords {
			if strings.Contains(lower, kw) {
				return warn(CheckDeprecatedCommands,
					fmt.Sprintf("command %q appears deprecated but no ## Deprecated section exists", cmd.Name))
			}
		}
	}
	return pass(CheckDeprecatedCommands, "no deprecated commands detected")
}

// externalDependencyKeywords indicate a tool has external dependencies.
var externalDependencyKeywords = []string{
	"api", "http", "network", "database", "service", "remote",
	"webhook", "endpoint", "cluster", "server",
}

// checkFailureModes recommends a Failure Modes section for complex or networked tools.
func checkFailureModes(sf *skillmd.SkillFile) CheckResult {
	if sf.Sections[skillmd.SectionFailureModes] != nil {
		return pass(CheckFailureModes, "Failure Modes section found")
	}
	if len(sf.Commands) >= 3 {
		return warn(CheckFailureModes, "3+ commands documented; consider adding ## Failure Modes section")
	}
	for _, cmd := range sf.Commands {
		lower := strings.ToLower(cmd.Desc)
		for _, kw := range externalDependencyKeywords {
			if strings.Contains(lower, kw) {
				return warn(CheckFailureModes,
					fmt.Sprintf("command %q references %q; consider adding ## Failure Modes section", cmd.Name, kw))
			}
		}
	}
	return pass(CheckFailureModes, "Failure Modes section not needed")
}

// checkDuplicateSkillNames warns on duplicate skill names across agent configs.
// This requires scanning the repository for multiple SKILL.md files.
func checkDuplicateSkillNames(repoPath string) CheckResult {
	if repoPath == "" {
		return pass(CheckDuplicateSkillNames, "duplicate check requires repo path")
	}

	// Find all SKILL.md files in the repo.
	var skillFiles []string
	var skillNames []string

	// Check root.
	rootSkill := filepath.Join(repoPath, "SKILL.md")
	if _, err := os.Stat(rootSkill); err == nil {
		skillFiles = append(skillFiles, rootSkill)
	}

	// Check docs/.
	docsSkill := filepath.Join(repoPath, "docs", "SKILL.md")
	if _, err := os.Stat(docsSkill); err == nil {
		skillFiles = append(skillFiles, docsSkill)
	}

	// Check for agent-specific SKILL.md files (e.g., .claude/SKILL.md, .qwen/SKILL.md).
	agentDirs := []string{".claude", ".qwen", ".cline", ".cursor", ".codex", ".opencode"}
	for _, dir := range agentDirs {
		agentSkill := filepath.Join(repoPath, dir, "SKILL.md")
		if _, err := os.Stat(agentSkill); err == nil {
			skillFiles = append(skillFiles, agentSkill)
		}
	}

	// Parse all found SKILL.md files and collect names.
	nameToPath := make(map[string]string)
	for _, sfPath := range skillFiles {
		sf, err := skillmd.ParseFile(sfPath)
		if err != nil {
			continue // Skip files that can't be parsed.
		}
		if sf.Name != "" {
			if existing, ok := nameToPath[sf.Name]; ok {
				rel1, _ := filepath.Rel(repoPath, existing)
				rel2, _ := filepath.Rel(repoPath, sfPath)
				skillNames = append(skillNames, fmt.Sprintf("%s (%s, %s)", sf.Name, rel1, rel2))
			} else {
				nameToPath[sf.Name] = sfPath
			}
		}
	}

	if len(skillNames) > 0 {
		return warn(CheckDuplicateSkillNames, fmt.Sprintf("duplicate skill names: %v", skillNames))
	}
	return pass(CheckDuplicateSkillNames, "no duplicate skill names found")
}

// --- Temporal contract checks ---

// changelogVersionPattern matches ## [x.y.z] headers in CHANGELOG.md.
var changelogVersionPattern = regexp.MustCompile(`^##\s+\[(\d+\.\d+\.\d+)\]`)

// checkChangelogExists verifies CHANGELOG.md exists at repo root.
func checkChangelogExists(repoPath string) CheckResult {
	p := filepath.Join(repoPath, "CHANGELOG.md")
	if _, err := os.Stat(p); err == nil {
		return pass(CheckChangelogExists, "CHANGELOG.md found")
	}
	return warn(CheckChangelogExists, "CHANGELOG.md not found at repo root")
}

// checkChangelogVersionEntry verifies the latest git tag has a matching CHANGELOG entry.
// Returns pass if no tags exist or if the repo is not a git repo.
func checkChangelogVersionEntry(repoPath string) CheckResult {
	changelogPath := filepath.Join(repoPath, "CHANGELOG.md")
	data, err := os.ReadFile(changelogPath)
	if err != nil {
		return pass(CheckChangelogVersionEntry, "no CHANGELOG.md to check")
	}

	// Parse version entries from CHANGELOG.
	var versions []string
	for _, line := range strings.Split(string(data), "\n") {
		if m := changelogVersionPattern.FindStringSubmatch(line); len(m) > 1 {
			versions = append(versions, m[1])
		}
	}

	if len(versions) == 0 {
		return fail(CheckChangelogVersionEntry, "CHANGELOG.md has no version entries (expected ## [x.y.z] headers)")
	}

	// Try to read latest git tag for cross-reference.
	latestTag := gitLatestTag(repoPath)
	if latestTag == "" {
		return pass(CheckChangelogVersionEntry, fmt.Sprintf("%d version(s) in CHANGELOG (no git tags to cross-reference)", len(versions)))
	}

	// Strip v prefix from tag.
	tagVersion := strings.TrimPrefix(latestTag, "v")

	// Check if the tag version exists in CHANGELOG.
	for _, v := range versions {
		if v == tagVersion {
			return pass(CheckChangelogVersionEntry, fmt.Sprintf("CHANGELOG entry found for %s", latestTag))
		}
	}
	return fail(CheckChangelogVersionEntry, fmt.Sprintf("no CHANGELOG entry for latest tag %s", latestTag))
}

// gitLatestTag returns the latest semver tag, or "" if none found.
func gitLatestTag(repoPath string) string {
	// Check if .git exists.
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); err != nil {
		return ""
	}
	// Use git describe to find latest tag.
	out, err := execGit(repoPath, "describe", "--tags", "--abbrev=0")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// execGit runs a git command in the given directory.
func execGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// --- Doctor provenance check ---

// checkDoctorProvenance verifies that doctor JSON output includes provenance fields.
func checkDoctorProvenance(sf *skillmd.SkillFile) CheckResult {
	for _, cmd := range sf.Commands {
		if !strings.HasSuffix(cmd.Name, " doctor") && cmd.Name != "doctor" {
			continue
		}
		if cmd.JSONOutput == "" {
			return pass(CheckDoctorProvenance, "doctor command has no JSON example (optional)")
		}
		var doc map[string]interface{}
		if err := json.Unmarshal([]byte(cmd.JSONOutput), &doc); err != nil {
			return pass(CheckDoctorProvenance, "doctor JSON not parseable (checked elsewhere)")
		}
		if _, ok := doc["version"]; !ok {
			return warn(CheckDoctorProvenance, "doctor output missing 'version' field (required for binary provenance)")
		}
		if _, ok := doc["source"]; !ok {
			return warn(CheckDoctorProvenance, "doctor output missing 'source' field (recommended: {repo: \"...\"})")
		}
		return pass(CheckDoctorProvenance, "doctor output includes provenance fields")
	}
	return pass(CheckDoctorProvenance, "no doctor command (checked elsewhere)")
}
