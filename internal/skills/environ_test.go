package skills

import (
	"fmt"
	"os"
	"testing"
)

// mockReadDir returns a function that simulates os.ReadDir behavior.
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

// mockStat returns a function that simulates os.Stat behavior.
func mockStat(behavior map[string]string) func(string) (os.FileInfo, error) {
	return func(path string) (os.FileInfo, error) {
		b, ok := behavior[path]
		if !ok {
			return nil, os.ErrNotExist
		}
		switch b {
		case "accessible":
			return nil, nil
		case "permission":
			return nil, fmt.Errorf("stat %s: %w", path, os.ErrPermission)
		case "notexist":
			return nil, os.ErrNotExist
		default:
			return nil, fmt.Errorf("unknown error")
		}
	}
}

func darwinTestEnv(dirBehavior map[string]string, fileBehavior map[string]string) *auditEnv {
	return &auditEnv{
		homeDir: "/home/user",
		readDir: mockReadDir(dirBehavior),
		stat:    mockStat(fileBehavior),
		goos:    "darwin",
	}
}

// --- sensitiveDirsForOS ---

func TestSensitiveDirsForOS_Darwin(t *testing.T) {
	dirs := sensitiveDirsForOS("darwin")
	if len(dirs) != 7 {
		t.Fatalf("darwin: got %d dirs, want 7", len(dirs))
	}
	names := make(map[string]bool)
	for _, d := range dirs {
		names[d.relPath] = true
	}
	if !names["Library"] {
		t.Error("darwin: missing Library")
	}
	if !names["Movies"] {
		t.Error("darwin: missing Movies")
	}
	if names["Videos"] {
		t.Error("darwin: should not have Videos")
	}
}

func TestSensitiveDirsForOS_Linux(t *testing.T) {
	dirs := sensitiveDirsForOS("linux")
	if len(dirs) != 6 {
		t.Fatalf("linux: got %d dirs, want 6", len(dirs))
	}
	names := make(map[string]bool)
	for _, d := range dirs {
		names[d.relPath] = true
	}
	if !names["Videos"] {
		t.Error("linux: missing Videos")
	}
	if names["Library"] {
		t.Error("linux: should not have Library")
	}
	if names["Movies"] {
		t.Error("linux: should not have Movies")
	}
}

func TestSensitiveDirsForOS_Windows(t *testing.T) {
	dirs := sensitiveDirsForOS("windows")
	if len(dirs) != 6 {
		t.Fatalf("windows: got %d dirs, want 6", len(dirs))
	}
	names := make(map[string]bool)
	for _, d := range dirs {
		names[d.relPath] = true
	}
	if !names["Videos"] {
		t.Error("windows: missing Videos")
	}
}

// --- auditEnvironment (directory checks) ---

func TestAuditEnvironment_AllBlocked(t *testing.T) {
	dirBehavior := map[string]string{
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
		"/home/user/.docker":   "permission",
		"/home/user/.kube":     "permission",
		"/home/user/.azure":    "permission",
		"/home/user/.gcloud":   "permission",
	}
	fileBehavior := map[string]string{
		"/home/user/.bash_history":           "permission",
		"/home/user/.zsh_history":            "permission",
		"/home/user/.sh_history":             "permission",
		"/home/user/.python_history":         "permission",
		"/home/user/.node_repl_history":      "permission",
		"/home/user/.netrc":                  "permission",
		"/home/user/.git-credentials":        "permission",
		"/home/user/.npmrc":                  "permission",
		"/home/user/.pypirc":                 "permission",
		"/home/user/.gem/credentials":        "permission",
		"/home/user/.cargo/credentials.toml": "permission",
	}

	env := darwinTestEnv(dirBehavior, fileBehavior)
	entries := auditEnvironment(env)

	if len(entries) != 25 {
		t.Fatalf("got %d entries, want 25", len(entries))
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
	dirBehavior := map[string]string{
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
		"/home/user/.docker":   "accessible",
		"/home/user/.kube":     "accessible",
		"/home/user/.azure":    "accessible",
		"/home/user/.gcloud":   "accessible",
	}
	fileBehavior := map[string]string{
		"/home/user/.bash_history":           "accessible",
		"/home/user/.zsh_history":            "accessible",
		"/home/user/.sh_history":             "accessible",
		"/home/user/.python_history":         "accessible",
		"/home/user/.node_repl_history":      "accessible",
		"/home/user/.netrc":                  "accessible",
		"/home/user/.git-credentials":        "accessible",
		"/home/user/.npmrc":                  "accessible",
		"/home/user/.pypirc":                 "accessible",
		"/home/user/.gem/credentials":        "accessible",
		"/home/user/.cargo/credentials.toml": "accessible",
	}

	env := darwinTestEnv(dirBehavior, fileBehavior)
	entries := auditEnvironment(env)

	warnCount := 0
	for _, e := range entries {
		if e.Status == AuditWarn {
			warnCount++
		}
	}
	if warnCount != 25 {
		t.Errorf("got %d warnings, want 25", warnCount)
	}
}

func TestAuditEnvironment_Mixed(t *testing.T) {
	dirBehavior := map[string]string{
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
		"/home/user/.docker":   "accessible",
		"/home/user/.kube":     "notexist",
		"/home/user/.azure":    "notexist",
		"/home/user/.gcloud":   "notexist",
	}
	fileBehavior := map[string]string{
		"/home/user/.zsh_history": "accessible",
		"/home/user/.npmrc":       "accessible",
	}

	env := darwinTestEnv(dirBehavior, fileBehavior)
	entries := auditEnvironment(env)

	if len(entries) != 25 {
		t.Fatalf("got %d entries, want 25", len(entries))
	}

	warnCount := 0
	okCount := 0
	for _, e := range entries {
		switch e.Status {
		case AuditWarn:
			warnCount++
		case AuditOK:
			okCount++
		}
	}
	// Downloads, .ssh, .docker (dirs) + .zsh_history, .npmrc (files) = 5 warnings
	if warnCount != 5 {
		t.Errorf("got %d warnings, want 5", warnCount)
	}
	if okCount != 20 {
		t.Errorf("got %d ok, want 20", okCount)
	}
}

func TestAuditEnvironment_NotExist(t *testing.T) {
	env := darwinTestEnv(map[string]string{}, map[string]string{})
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
		stat:    os.Stat,
		goos:    "darwin",
	}
	entries := auditEnvironment(env)
	if entries != nil {
		t.Errorf("expected nil, got %d entries", len(entries))
	}
}

func TestAuditEnvironment_Linux(t *testing.T) {
	env := &auditEnv{
		homeDir: "/home/user",
		readDir: mockReadDir(map[string]string{}),
		stat:    mockStat(map[string]string{}),
		goos:    "linux",
	}
	entries := auditEnvironment(env)
	// Linux: 6 sensitive + 7 credential + 5 history + 6 credential-file = 24
	if len(entries) != 24 {
		t.Fatalf("linux: got %d entries, want 24", len(entries))
	}
}

// --- probeDir ---

func TestProbeDir_Categories(t *testing.T) {
	env := &auditEnv{
		homeDir: "/home/user",
		readDir: mockReadDir(map[string]string{
			"/home/user/Documents": "accessible",
			"/home/user/.ssh":      "accessible",
		}),
		stat: mockStat(map[string]string{}),
		goos: "darwin",
	}

	sensEntry := probeDir(sensitiveDirsForOS("darwin")[0], env) // ~/Documents
	if sensEntry.Category != "sensitive-dir" {
		t.Errorf("category = %q, want sensitive-dir", sensEntry.Category)
	}

	credEntry := probeDir(credentialDirChecks[0], env) // ~/.ssh
	if credEntry.Category != "credential-dir" {
		t.Errorf("category = %q, want credential-dir", credEntry.Category)
	}
}

// --- probeFile ---

func TestProbeFile_Accessible(t *testing.T) {
	env := &auditEnv{
		homeDir: "/home/user",
		stat: mockStat(map[string]string{
			"/home/user/.bash_history": "accessible",
		}),
	}
	entry := probeFile(historyFileChecks[0], env)
	if entry.Status != AuditWarn {
		t.Errorf("status = %s, want warn", entry.Status)
	}
	if entry.Category != "history-file" {
		t.Errorf("category = %q, want history-file", entry.Category)
	}
}

func TestProbeFile_NotExist(t *testing.T) {
	env := &auditEnv{
		homeDir: "/home/user",
		stat:    mockStat(map[string]string{}),
	}
	entry := probeFile(historyFileChecks[0], env)
	if entry.Status != AuditOK {
		t.Errorf("status = %s, want ok", entry.Status)
	}
	if entry.Message != "not present" {
		t.Errorf("message = %q, want 'not present'", entry.Message)
	}
}

func TestProbeFile_Permission(t *testing.T) {
	env := &auditEnv{
		homeDir: "/home/user",
		stat: mockStat(map[string]string{
			"/home/user/.bash_history": "permission",
		}),
	}
	entry := probeFile(historyFileChecks[0], env)
	if entry.Status != AuditOK {
		t.Errorf("status = %s, want ok", entry.Status)
	}
	if entry.Message != "blocked (access denied)" {
		t.Errorf("message = %q, want 'blocked (access denied)'", entry.Message)
	}
}

func TestProbeFile_Categories(t *testing.T) {
	env := &auditEnv{
		homeDir: "/home/user",
		stat: mockStat(map[string]string{
			"/home/user/.bash_history": "accessible",
			"/home/user/.netrc":        "accessible",
		}),
	}

	histEntry := probeFile(historyFileChecks[0], env)
	if histEntry.Category != "history-file" {
		t.Errorf("category = %q, want history-file", histEntry.Category)
	}

	credEntry := probeFile(credentialFileChecks[0], env)
	if credEntry.Category != "credential-file" {
		t.Errorf("category = %q, want credential-file", credEntry.Category)
	}
}
