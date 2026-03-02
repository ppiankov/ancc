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

// fileCheck describes a file to probe for existence.
type fileCheck struct {
	category string // "history-file" or "credential-file"
	relPath  string // path relative to home, e.g. ".bash_history"
	label    string // display label, e.g. "~/.bash_history"
	warnMsg  string // message when accessible
}

// sensitiveDirsForOS returns platform-appropriate sensitive directory checks.
func sensitiveDirsForOS(goos string) []dirCheck {
	common := []dirCheck{
		{"sensitive-dir", "Documents", "~/Documents", "accessible (agents can read this directory)"},
		{"sensitive-dir", "Downloads", "~/Downloads", "accessible (agents can read this directory)"},
		{"sensitive-dir", "Desktop", "~/Desktop", "accessible (agents can read this directory)"},
		{"sensitive-dir", "Pictures", "~/Pictures", "accessible (agents can read this directory)"},
		{"sensitive-dir", "Music", "~/Music", "accessible (agents can read this directory)"},
	}

	switch goos {
	case "darwin":
		return append(common,
			dirCheck{"sensitive-dir", "Movies", "~/Movies", "accessible (agents can read this directory)"},
			dirCheck{"sensitive-dir", "Library", "~/Library", "accessible (agents can read this directory)"},
		)
	default: // linux, windows
		return append(common,
			dirCheck{"sensitive-dir", "Videos", "~/Videos", "accessible (agents can read this directory)"},
		)
	}
}

var credentialDirChecks = []dirCheck{
	{"credential-dir", ".ssh", "~/.ssh", "accessible (contains credentials, agents can read)"},
	{"credential-dir", ".aws", "~/.aws", "accessible (contains credentials, agents can read)"},
	{"credential-dir", ".gnupg", "~/.gnupg", "accessible (contains credentials, agents can read)"},
	{"credential-dir", ".docker", "~/.docker", "accessible (contains credentials, agents can read)"},
	{"credential-dir", ".kube", "~/.kube", "accessible (contains credentials, agents can read)"},
	{"credential-dir", ".azure", "~/.azure", "accessible (contains credentials, agents can read)"},
	{"credential-dir", ".gcloud", "~/.gcloud", "accessible (contains credentials, agents can read)"},
}

var historyFileChecks = []fileCheck{
	{"history-file", ".bash_history", "~/.bash_history", "accessible (may contain accidentally typed secrets)"},
	{"history-file", ".zsh_history", "~/.zsh_history", "accessible (may contain accidentally typed secrets)"},
	{"history-file", ".sh_history", "~/.sh_history", "accessible (may contain accidentally typed secrets)"},
	{"history-file", ".python_history", "~/.python_history", "accessible (may contain accidentally typed secrets)"},
	{"history-file", ".node_repl_history", "~/.node_repl_history", "accessible (may contain accidentally typed secrets)"},
}

var credentialFileChecks = []fileCheck{
	{"credential-file", ".netrc", "~/.netrc", "accessible (contains plaintext credentials)"},
	{"credential-file", ".git-credentials", "~/.git-credentials", "accessible (contains plaintext credentials)"},
	{"credential-file", ".npmrc", "~/.npmrc", "accessible (may contain auth tokens)"},
	{"credential-file", ".pypirc", "~/.pypirc", "accessible (contains plaintext credentials)"},
	{"credential-file", ".gem/credentials", "~/.gem/credentials", "accessible (contains plaintext credentials)"},
	{"credential-file", ".cargo/credentials.toml", "~/.cargo/credentials.toml", "accessible (contains plaintext credentials)"},
}

// auditEnvironment probes sensitive and credential directories, history files,
// and credential files for accessibility.
func auditEnvironment(env *auditEnv) []AuditEntry {
	if env.homeDir == "" {
		return nil
	}

	var entries []AuditEntry

	for _, check := range sensitiveDirsForOS(env.goos) {
		entries = append(entries, probeDir(check, env))
	}
	for _, check := range credentialDirChecks {
		entries = append(entries, probeDir(check, env))
	}
	for _, check := range historyFileChecks {
		entries = append(entries, probeFile(check, env))
	}
	for _, check := range credentialFileChecks {
		entries = append(entries, probeFile(check, env))
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

// probeFile attempts to stat a file and returns an AuditEntry based on the result.
func probeFile(check fileCheck, env *auditEnv) AuditEntry {
	path := filepath.Join(env.homeDir, check.relPath)

	_, err := env.stat(path)
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
