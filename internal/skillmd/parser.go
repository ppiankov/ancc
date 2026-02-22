package skillmd

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	reHeading   = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
	reBoldLabel = regexp.MustCompile(`^\*\*(.+?):\*\*`)
	reFlag      = regexp.MustCompile("^-\\s+`(--[^`]+)`\\s*(?:—|-)\\s*(.+)$")
	reExitCode  = regexp.MustCompile(`^-\s+(\d+):\s*(.+)$`)
)

// ParseFile reads a SKILL.md file from disk and parses it.
func ParseFile(path string) (*SkillFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading skill file: %w", err)
	}
	return Parse(string(data))
}

// Parse parses SKILL.md content into a structured representation.
func Parse(content string) (*SkillFile, error) {
	lines := strings.Split(content, "\n")
	sf := &SkillFile{
		Sections: make(map[string]*Section),
	}

	// Extract H1 name and description.
	i := parseHeader(lines, sf)

	// Split remaining lines into H2 sections.
	parseSections(lines[i:], sf)

	// Extract commands from the Commands section.
	if cmdSection, ok := sf.Sections[SectionCommands]; ok {
		sf.Commands = parseCommands(cmdSection.Content)
	}

	return sf, nil
}

// parseHeader extracts the H1 heading and first paragraph as description.
// Returns the line index after the description.
func parseHeader(lines []string, sf *SkillFile) int {
	i := 0

	// Skip leading blank lines.
	for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		i++
	}

	// Look for H1.
	if i < len(lines) {
		if m := reHeading.FindStringSubmatch(lines[i]); m != nil && len(m[1]) == 1 {
			sf.Name = strings.TrimSpace(m[2])
			i++
		}
	}

	// Skip blank lines after H1.
	for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		i++
	}

	// Collect description paragraph (lines until blank line or next heading).
	var desc []string
	for i < len(lines) {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			break
		}
		if reHeading.MatchString(line) {
			break
		}
		desc = append(desc, line)
		i++
	}
	sf.Description = strings.Join(desc, " ")

	return i
}

// parseSections splits lines into H2 sections and adds them to sf.
func parseSections(lines []string, sf *SkillFile) {
	var currentHeading string
	var currentLines []string

	flush := func() {
		if currentHeading != "" {
			sf.Sections[currentHeading] = &Section{
				Heading: currentHeading,
				Level:   2,
				Content: strings.TrimSpace(strings.Join(currentLines, "\n")),
			}
		}
	}

	for _, line := range lines {
		if m := reHeading.FindStringSubmatch(line); m != nil && len(m[1]) == 2 {
			flush()
			currentHeading = strings.TrimSpace(m[2])
			currentLines = nil
			continue
		}
		currentLines = append(currentLines, line)
	}
	flush()
}

// parseCommands extracts Command definitions from H3 headings within the Commands section.
func parseCommands(content string) []Command {
	lines := strings.Split(content, "\n")
	var commands []Command
	var current *Command

	flush := func() {
		if current != nil {
			commands = append(commands, *current)
		}
	}

	for i := 0; i < len(lines); i++ {
		if m := reHeading.FindStringSubmatch(lines[i]); m != nil && len(m[1]) == 3 {
			flush()
			current = &Command{Name: strings.TrimSpace(m[2])}
			continue
		}

		if current == nil {
			continue
		}

		line := lines[i]

		// Check for description (first non-empty, non-label line after heading).
		if current.Desc == "" && strings.TrimSpace(line) != "" && !reBoldLabel.MatchString(line) {
			current.Desc = strings.TrimSpace(line)
			continue
		}

		// Check for bold labels.
		if bm := reBoldLabel.FindStringSubmatch(line); bm != nil {
			label := strings.TrimRight(bm[1], ":")
			switch label {
			case SubsectionFlags:
				i = parseFlags(lines, i+1, current)
			case SubsectionJSONOutput:
				i = parseJSONOutput(lines, i+1, current)
			case SubsectionExitCodes:
				i = parseExitCodes(lines, i+1, current)
			}
		}
	}
	flush()

	return commands
}

// parseFlags extracts flag definitions from list items starting at line i.
func parseFlags(lines []string, i int, cmd *Command) int {
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			return i
		}
		if fm := reFlag.FindStringSubmatch(line); fm != nil {
			cmd.Flags = append(cmd.Flags, Flag{Name: fm[1], Desc: fm[2]})
		} else if !strings.HasPrefix(line, "-") {
			return i - 1
		}
		i++
	}
	return i - 1
}

// parseJSONOutput extracts the JSON code block.
func parseJSONOutput(lines []string, i int, cmd *Command) int {
	// Find opening code fence.
	for i < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
		i++
	}
	if i >= len(lines) {
		return i - 1
	}
	i++ // skip opening fence

	var block []string
	for i < len(lines) {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
			cmd.JSONOutput = strings.Join(block, "\n")
			return i
		}
		block = append(block, lines[i])
		i++
	}
	return i - 1
}

// parseExitCodes extracts exit code definitions from list items.
func parseExitCodes(lines []string, i int, cmd *Command) int {
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			return i
		}
		if em := reExitCode.FindStringSubmatch(line); em != nil {
			code, _ := strconv.Atoi(em[1])
			cmd.ExitCodes = append(cmd.ExitCodes, ExitCode{Code: code, Desc: em[2]})
		} else if !strings.HasPrefix(line, "-") {
			return i - 1
		}
		i++
	}
	return i - 1
}
