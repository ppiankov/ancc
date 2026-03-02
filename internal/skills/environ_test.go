package skills

import (
	"fmt"
	"os"
	"testing"
)

// mockReadDir returns a function that simulates os.ReadDir behavior.
// accessible: returns nil error (directory readable)
// permission: returns os.ErrPermission
// notexist: returns os.ErrNotExist
func mockReadDir(behavior map[string]string) func(string) ([]os.DirEntry, error) {
	return func(path string) ([]os.DirEntry, error) {
		b, ok := behavior[path]
		if !ok {
			return nil, os.ErrNotExist
		}
		switch b {
		case "accessible":
			return []os.DirEntry{}, nil
		case "permission":
			return nil, fmt.Errorf("open %s: %w", path, os.ErrPermission)
		case "notexist":
			return nil, os.ErrNotExist
		default:
			return nil, fmt.Errorf("unknown error")
		}
	}
}

func TestAuditEnvironment_AllBlocked(t *testing.T) {
	behavior := map[string]string{
		"/home/user/Documents": "permission",
		"/home/user/Downloads": "permission",
		"/home/user/Desktop":   "permission",
		"/home/user/Pictures":  "permission",
		"/home/user/Music":     "permission",
		"/home/user/Movies":    "permission",
		"/home/user/Library":   "permission",
		"/home/user/.ssh":      "permission",
		"/home/user/.aws":      "permission",
		"/home/user/.gnupg":    "permission",
	}

	env := &auditEnv{
		homeDir: "/home/user",
		readDir: mockReadDir(behavior),
	}

	entries := auditEnvironment(env)
	if len(entries) != 10 {
		t.Fatalf("got %d entries, want 10", len(entries))
	}

	for _, e := range entries {
		if e.Status != AuditOK {
			t.Errorf("%s: status = %s, want ok", e.Name, e.Status)
		}
		if e.Message != "blocked (access denied)" {
			t.Errorf("%s: message = %q, want 'blocked (access denied)'", e.Name, e.Message)
		}
	}
}

func TestAuditEnvironment_AllAccessible(t *testing.T) {
	behavior := map[string]string{
		"/home/user/Documents": "accessible",
		"/home/user/Downloads": "accessible",
		"/home/user/Desktop":   "accessible",
		"/home/user/Pictures":  "accessible",
		"/home/user/Music":     "accessible",
		"/home/user/Movies":    "accessible",
		"/home/user/Library":   "accessible",
		"/home/user/.ssh":      "accessible",
		"/home/user/.aws":      "accessible",
		"/home/user/.gnupg":    "accessible",
	}

	env := &auditEnv{
		homeDir: "/home/user",
		readDir: mockReadDir(behavior),
	}

	entries := auditEnvironment(env)

	warnCount := 0
	for _, e := range entries {
		if e.Status == AuditWarn {
			warnCount++
		}
	}
	if warnCount != 10 {
		t.Errorf("got %d warnings, want 10", warnCount)
	}
}

func TestAuditEnvironment_Mixed(t *testing.T) {
	behavior := map[string]string{
		"/home/user/Documents": "permission",
		"/home/user/Downloads": "accessible",
		"/home/user/Desktop":   "permission",
		"/home/user/Pictures":  "notexist",
		"/home/user/Music":     "notexist",
		"/home/user/Movies":    "notexist",
		"/home/user/Library":   "permission",
		"/home/user/.ssh":      "accessible",
		"/home/user/.aws":      "notexist",
		"/home/user/.gnupg":    "permission",
	}

	env := &auditEnv{
		homeDir: "/home/user",
		readDir: mockReadDir(behavior),
	}

	entries := auditEnvironment(env)
	if len(entries) != 10 {
		t.Fatalf("got %d entries, want 10", len(entries))
	}

	okCount := 0
	warnCount := 0
	for _, e := range entries {
		switch e.Status {
		case AuditOK:
			okCount++
		case AuditWarn:
			warnCount++
		}
	}
	if warnCount != 2 { // ~/Downloads + ~/.ssh
		t.Errorf("got %d warnings, want 2", warnCount)
	}
	if okCount != 8 {
		t.Errorf("got %d ok, want 8", okCount)
	}
}

func TestAuditEnvironment_NotExist(t *testing.T) {
	// All directories don't exist (default behavior of mockReadDir for unknown paths).
	env := &auditEnv{
		homeDir: "/home/user",
		readDir: mockReadDir(map[string]string{}),
	}

	entries := auditEnvironment(env)
	for _, e := range entries {
		if e.Status != AuditOK {
			t.Errorf("%s: status = %s, want ok", e.Name, e.Status)
		}
		if e.Message != "not present" {
			t.Errorf("%s: message = %q, want 'not present'", e.Name, e.Message)
		}
	}
}

func TestAuditEnvironment_NoHome(t *testing.T) {
	env := &auditEnv{
		homeDir: "",
		readDir: os.ReadDir,
	}

	entries := auditEnvironment(env)
	if entries != nil {
		t.Errorf("expected nil, got %d entries", len(entries))
	}
}

func TestProbeDir_Categories(t *testing.T) {
	env := &auditEnv{
		homeDir: "/home/user",
		readDir: mockReadDir(map[string]string{
			"/home/user/Documents": "accessible",
			"/home/user/.ssh":      "accessible",
		}),
	}

	sensEntry := probeDir(sensitiveDirChecks[0], env) // ~/Documents
	if sensEntry.Category != "sensitive-dir" {
		t.Errorf("category = %q, want sensitive-dir", sensEntry.Category)
	}

	credEntry := probeDir(credentialDirChecks[0], env) // ~/.ssh
	if credEntry.Category != "credential-dir" {
		t.Errorf("category = %q, want credential-dir", credEntry.Category)
	}
}
