package validator

// Status values for individual checks.
const (
	StatusPass = "pass"
	StatusFail = "fail"
	StatusWarn = "warn"
)

// Overall validation status values.
const (
	OverallPass    = "pass"
	OverallFail    = "fail"
	OverallPartial = "partial"
)

// CheckResult holds the outcome of a single validation check.
type CheckResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "pass", "fail", "warn"
	Message string `json:"message"`
}

// Summary holds aggregated counts.
type Summary struct {
	Total int `json:"total"`
	Pass  int `json:"pass"`
	Fail  int `json:"fail"`
	Warn  int `json:"warn"`
}

// ValidationResult holds the full validation outcome.
type ValidationResult struct {
	Path    string        `json:"path"`
	Status  string        `json:"status"` // "pass", "fail", "partial"
	Checks  []CheckResult `json:"checks"`
	Summary Summary       `json:"summary"`
}
