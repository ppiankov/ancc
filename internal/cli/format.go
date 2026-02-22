package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/ppiankov/ancc/internal/validator"
)

// Check name to human-readable label mapping.
var checkLabels = map[string]string{
	validator.CheckSkillMDExists:    "SKILL.md exists",
	validator.CheckSkillMDInstall:   "Install section",
	validator.CheckSkillMDCommands:  "Commands section",
	validator.CheckSkillMDFlags:     "Flags documented",
	validator.CheckSkillMDJSON:      "JSON output schema",
	validator.CheckSkillMDExitCodes: "Exit codes documented",
	validator.CheckSkillMDNotDo:     "What this does NOT do",
	validator.CheckSkillMDParsing:   "Parsing examples",
	validator.CheckHasInitCommand:   "Init command",
	validator.CheckHasDoctorCommand: "Doctor command",
	validator.CheckHasBinaryRelease: "Binary release",
}

const labelWidth = 35

func formatText(w io.Writer, result *validator.ValidationResult, verbose bool) {
	for _, c := range result.Checks {
		if !verbose && c.Status == validator.StatusPass {
			continue
		}

		label := checkLabels[c.Name]
		if label == "" {
			label = c.Name
		}

		dots := labelWidth - len(label)
		if dots < 3 {
			dots = 3
		}

		status := strings.ToUpper(c.Status)
		line := fmt.Sprintf("  %s %s %s", label, strings.Repeat(".", dots), status)

		if c.Status != validator.StatusPass && c.Message != "" {
			line += "  " + c.Message
		}

		_, _ = fmt.Fprintln(w, line)
	}

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "  Result: %s (%d pass, %d fail, %d warn)\n",
		strings.ToUpper(result.Status),
		result.Summary.Pass,
		result.Summary.Fail,
		result.Summary.Warn,
	)
}

func formatJSON(w io.Writer, result *validator.ValidationResult) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}
