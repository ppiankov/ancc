package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/ppiankov/ancc/internal/skills"
	"github.com/spf13/cobra"
)

// DoctorCheck holds the result of a single health check.
type DoctorCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// DoctorResult holds all health check results.
type DoctorResult struct {
	Status string               `json:"status"`
	Checks []DoctorCheck        `json:"checks"`
	Agents []DoctorAgentPosture `json:"agents,omitempty"` // WO-78: per-agent enforcement posture.
}

// DoctorAgentPosture mirrors scanner enforcement without changing doctor status.
//
// WO-78: doctor exposes posture evidence without failing advisory agents.
type DoctorAgentPosture struct {
	Name                string             `json:"name"`
	Enforcement         skills.Enforcement `json:"enforcement"`
	EnforcementEvidence string             `json:"enforcement_evidence,omitempty"`
	Caution             string             `json:"caution,omitempty"`
	Mitigation          string             `json:"mitigation,omitempty"`
}

const (
	doctorOK    = "ok"
	doctorWarn  = "warn"
	doctorError = "error"
)

// doctorEnv holds injectable dependencies for testing.
type doctorEnv struct {
	ctx          context.Context
	version      string
	githubAPIURL string
	lookPath     func(string) (string, error)
	httpDo       func(*http.Request) (*http.Response, error)
	getenv       func(string) string
	cmdOutput    func(string, ...string) ([]byte, error)
	projectDir   string
	scanAgents   func(string) (*skills.ScanResult, error)
}

func defaultDoctorEnv(ctx context.Context, version string) *doctorEnv {
	if ctx == nil {
		ctx = context.Background()
	}

	httpDo := func(req *http.Request) (*http.Response, error) {
		client := &http.Client{}
		return client.Do(req)
	}
	return &doctorEnv{
		ctx:          ctx,
		version:      version,
		githubAPIURL: "https://api.github.com",
		lookPath:     exec.LookPath,
		httpDo:       httpDo,
		getenv:       os.Getenv,
		cmdOutput: func(name string, args ...string) ([]byte, error) {
			return exec.Command(name, args...).Output()
		},
		projectDir: ".",
		scanAgents: skills.Scan,
	}
}

func newDoctorCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check ancc health and environment",
		RunE: func(cmd *cobra.Command, _ []string) error {
			env := defaultDoctorEnv(cmd.Context(), cmd.Root().Version)
			result := runDoctor(env)

			w := cmd.OutOrStdout()
			switch format {
			case "json":
				if err := formatDoctorJSON(w, result); err != nil {
					return fmt.Errorf("formatting output: %w", err)
				}
			default:
				formatDoctorText(w, result)
			}

			if result.Status == doctorError {
				return &ExitError{Code: 1}
			}
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().StringVar(&format, "format", "text", "output format (text, json)")

	return cmd
}

func runDoctor(env *doctorEnv) *DoctorResult {
	result := &DoctorResult{}

	result.Checks = append(result.Checks, checkVersion(env))
	result.Checks = append(result.Checks, checkGo(env))
	result.Checks = append(result.Checks, checkGitHubAPI(env))
	result.Checks = append(result.Checks, checkHomebrew(env))
	result.Agents = checkAgentPosture(env)

	result.Status = doctorOK
	for _, c := range result.Checks {
		if c.Status == doctorError {
			result.Status = doctorError
			break
		}
		if c.Status == doctorWarn {
			result.Status = doctorWarn
		}
	}

	return result
}

// WO-78: surface posture as evidence for users, not as a health failure.
func checkAgentPosture(env *doctorEnv) []DoctorAgentPosture {
	if env == nil || env.scanAgents == nil {
		return nil
	}

	projectDir := env.projectDir
	if projectDir == "" {
		projectDir = "."
	}

	scanResult, err := env.scanAgents(projectDir)
	if err != nil || scanResult == nil {
		return nil
	}

	posture := make([]DoctorAgentPosture, 0, len(scanResult.Agents))
	for _, a := range scanResult.Agents {
		entry := DoctorAgentPosture{
			Name:                a.Name,
			Enforcement:         agentEnforcement(a),
			EnforcementEvidence: a.EnforcementEvidence,
		}
		if entry.Enforcement == skills.EnforcementAdvisory {
			entry.Caution = advisoryCaution
			entry.Mitigation = advisoryMitigation
		}
		posture = append(posture, entry)
	}
	return posture
}

func checkVersion(env *doctorEnv) DoctorCheck {
	v := env.version
	if v == "" {
		v = "dev"
	}
	return DoctorCheck{Name: "ancc-version", Status: doctorOK, Message: v}
}

func compareVersions(a, b string) int {
	aParts, aOK := parseVersion(a)
	bParts, bOK := parseVersion(b)
	if !aOK || !bOK {
		return strings.Compare(strings.TrimSpace(a), strings.TrimSpace(b))
	}

	for i := range aParts {
		switch {
		case aParts[i] < bParts[i]:
			return -1
		case aParts[i] > bParts[i]:
			return 1
		}
	}

	return 0
}

func parseVersion(v string) ([3]int, bool) {
	v = strings.TrimSpace(strings.TrimPrefix(v, "v"))
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return [3]int{}, false
	}

	var parsed [3]int
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil {
			return [3]int{}, false
		}
		parsed[i] = n
	}

	return parsed, true
}

func checkGo(env *doctorEnv) DoctorCheck {
	_, err := env.lookPath("go")
	if err != nil {
		return DoctorCheck{Name: "go-available", Status: doctorWarn, Message: "go not found in PATH"}
	}

	out, err := env.cmdOutput("go", "version")
	if err != nil {
		return DoctorCheck{Name: "go-available", Status: doctorWarn, Message: "go found but version check failed"}
	}

	version := strings.TrimSpace(string(out))
	return DoctorCheck{Name: "go-available", Status: doctorOK, Message: version}
}

func checkGitHubAPI(env *doctorEnv) DoctorCheck {
	token := env.getenv("GITHUB_TOKEN")
	if token == "" {
		return DoctorCheck{Name: "github-api", Status: doctorWarn, Message: "GITHUB_TOKEN not set"}
	}

	req, err := http.NewRequestWithContext(doctorContext(env), http.MethodGet, env.githubAPIURL, nil)
	if err != nil {
		return DoctorCheck{Name: "github-api", Status: doctorError, Message: "GitHub API request failed"}
	}

	resp, err := env.httpDo(req)
	if err != nil {
		return DoctorCheck{Name: "github-api", Status: doctorError, Message: "GitHub API unreachable"}
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return DoctorCheck{Name: "github-api", Status: doctorError, Message: fmt.Sprintf("GitHub API returned %d", resp.StatusCode)}
	}

	return DoctorCheck{Name: "github-api", Status: doctorOK, Message: "GitHub API reachable"}
}

func checkHomebrew(env *doctorEnv) DoctorCheck {
	_, err := env.lookPath("brew")
	if err != nil {
		return DoctorCheck{Name: "homebrew", Status: doctorWarn, Message: "brew not found in PATH"}
	}
	return DoctorCheck{Name: "homebrew", Status: doctorOK, Message: "brew found"}
}

func doctorContext(env *doctorEnv) context.Context {
	if env != nil && env.ctx != nil {
		return env.ctx
	}
	return context.Background()
}

const doctorLabelWidth = 35

func formatDoctorText(w io.Writer, result *DoctorResult) {
	for _, c := range result.Checks {
		label := doctorCheckLabels[c.Name]
		if label == "" {
			label = c.Name
		}

		dots := doctorLabelWidth - len(label)
		if dots < 3 {
			dots = 3
		}

		status := strings.ToUpper(c.Status)
		line := fmt.Sprintf("  %s %s %s", label, strings.Repeat(".", dots), status)
		if c.Message != "" {
			line += "  " + c.Message
		}
		_, _ = fmt.Fprintln(w, line)
	}

	if len(result.Agents) > 0 {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "  Agent enforcement:")
		_, _ = fmt.Fprintf(w, "  %-*s %-*s\n",
			skillsAgentWidth, "Agent",
			skillsEnforcementWidth, "Posture",
		)
		for _, a := range result.Agents {
			_, _ = fmt.Fprintf(w, "  %-*s %-*s\n",
				skillsAgentWidth, a.Name,
				skillsEnforcementWidth, a.Enforcement,
			)
			if a.Caution != "" {
				_, _ = fmt.Fprintf(w, "  %-*s caution: %s; evidence: %s; mitigation: %s\n",
					skillsAgentWidth, a.Name, a.Caution, a.EnforcementEvidence, a.Mitigation)
			}
		}
	}

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "  Result: %s\n", strings.ToUpper(result.Status))
}

func formatDoctorJSON(w io.Writer, result *DoctorResult) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

var doctorCheckLabels = map[string]string{
	"ancc-version": "ancc version",
	"go-available": "Go available",
	"github-api":   "GitHub API",
	"homebrew":     "Homebrew",
}
