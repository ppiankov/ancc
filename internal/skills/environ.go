package skills

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// dirCheck describes a directory to probe for access.
type dirCheck struct {
	category string // "sensitive-dir" or "credential-dir"
	relPath  string // path relative to home, e.g. "Documents"
	label    string // display label, e.g. "~/Documents"
	warnMsg  string // message when accessible
}

var sensitiveDirChecks = []dirCheck{
	{"sensitive-dir", "Documents", "~/Documents", "accessible (agents can read this directory)"},
	{"sensitive-dir", "Downloads", "~/Downloads", "accessible (agents can read this directory)"},
	{"sensitive-dir", "Desktop", "~/Desktop", "accessible (agents can read this directory)"},
	{"sensitive-dir", "Pictures", "~/Pictures", "accessible (agents can read this directory)"},
	{"sensitive-dir", "Music", "~/Music", "accessible (agents can read this directory)"},
	{"sensitive-dir", "Movies", "~/Movies", "accessible (agents can read this directory)"},
	{"sensitive-dir", "Library", "~/Library", "accessible (agents can read this directory)"},
}

var credentialDirChecks = []dirCheck{
	{"credential-dir", ".ssh", "~/.ssh", "accessible (contains credentials, agents can read)"},
	{"credential-dir", ".aws", "~/.aws", "accessible (contains credentials, agents can read)"},
	{"credential-dir", ".gnupg", "~/.gnupg", "accessible (contains credentials, agents can read)"},
}

// auditEnvironment probes sensitive and credential directories for accessibility.
func auditEnvironment(env *auditEnv) []AuditEntry {
	if env.homeDir == "" {
		return nil
	}

	var entries []AuditEntry
	for _, check := range sensitiveDirChecks {
		entries = append(entries, probeDir(check, env))
	}
	for _, check := range credentialDirChecks {
		entries = append(entries, probeDir(check, env))
	}
	return entries
}

// probeDir attempts to read a directory and returns an AuditEntry based on the result.
func probeDir(check dirCheck, env *auditEnv) AuditEntry {
	dir := filepath.Join(env.homeDir, check.relPath)

	_, err := env.readDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			return AuditEntry{
				Category: check.category,
				Name:     check.label,
				Status:   AuditOK,
				Message:  "blocked (access denied)",
			}
		}
		if errors.Is(err, os.ErrNotExist) {
			return AuditEntry{
				Category: check.category,
				Name:     check.label,
				Status:   AuditOK,
				Message:  "not present",
			}
		}
		return AuditEntry{
			Category: check.category,
			Name:     check.label,
			Status:   AuditOK,
			Message:  fmt.Sprintf("not accessible (%v)", err),
		}
	}

	return AuditEntry{
		Category: check.category,
		Name:     check.label,
		Status:   AuditWarn,
		Message:  check.warnMsg,
	}
}
