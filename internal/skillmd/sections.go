package skillmd

// Required section headings in SKILL.md.
const (
	SectionInstall         = "Install"
	SectionCommands        = "Commands"
	SectionWhatNotDo       = "What this does NOT do"
	SectionParsingExamples = "Parsing examples"
)

// Per-command subsections.
const (
	SubsectionFlags      = "Flags"
	SubsectionJSONOutput = "JSON output"
	SubsectionExitCodes  = "Exit codes"
)

// SkillFile represents a parsed SKILL.md.
type SkillFile struct {
	Name        string
	Description string
	Sections    map[string]*Section
	Commands    []Command
}

// Section represents a markdown section (H2).
type Section struct {
	Heading string
	Level   int
	Content string
}

// Command represents a documented CLI command (H3 under Commands).
type Command struct {
	Name       string
	Desc       string
	Flags      []Flag
	JSONOutput string
	ExitCodes  []ExitCode
}

// Flag represents a documented CLI flag.
type Flag struct {
	Name string
	Desc string
}

// ExitCode represents a documented exit code.
type ExitCode struct {
	Code int
	Desc string
}
