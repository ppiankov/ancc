package validator

import (
	"os"
	"path/filepath"
	"testing"
)

// makeRepo creates a fake git repo with an optional SKILL.md.
func makeRepo(t *testing.T, parent, name string, withSkillMD bool) string {
	t.Helper()
	dir := filepath.Join(parent, name)
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if withSkillMD {
		data, err := readFile(testdataPath("valid-skill.md"))
		if err != nil {
			t.Fatal(err)
		}
		if err := writeFile(filepath.Join(dir, "SKILL.md"), data); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestScanRepos_FindsRepos(t *testing.T) {
	root := t.TempDir()
	makeRepo(t, root, "alpha", true)
	makeRepo(t, root, "beta", true)

	result, err := ScanRepos(root, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Repos) != 2 {
		t.Fatalf("repos = %d, want 2", len(result.Repos))
	}
	if result.Summary.Total != 2 {
		t.Errorf("total = %d, want 2", result.Summary.Total)
	}
}

func TestScanRepos_MissingSkillMD(t *testing.T) {
	root := t.TempDir()
	makeRepo(t, root, "noskill", false)

	result, err := ScanRepos(root, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Repos) != 1 {
		t.Fatalf("repos = %d, want 1", len(result.Repos))
	}
	if result.Repos[0].Status != ScanStatusMissing {
		t.Errorf("status = %q, want %q", result.Repos[0].Status, ScanStatusMissing)
	}
	if result.Summary.Missing != 1 {
		t.Errorf("missing = %d, want 1", result.Summary.Missing)
	}
}

func TestScanRepos_SkipsHiddenDirs(t *testing.T) {
	root := t.TempDir()
	makeRepo(t, root, ".hidden", true)
	makeRepo(t, root, "visible", true)

	result, err := ScanRepos(root, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Repos) != 1 {
		t.Fatalf("repos = %d, want 1", len(result.Repos))
	}
	if result.Repos[0].Name != "visible" {
		t.Errorf("name = %q, want %q", result.Repos[0].Name, "visible")
	}
}

func TestScanRepos_DepthLimit(t *testing.T) {
	root := t.TempDir()
	// Repo at depth 1 — should be found.
	makeRepo(t, root, "shallow", true)
	// Repo at depth 3 — should not be found with depth=2.
	deep := filepath.Join(root, "a", "b", "deep")
	if err := os.MkdirAll(filepath.Join(deep, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := ScanRepos(root, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Repos) != 1 {
		t.Fatalf("repos = %d, want 1 (deep repo should be excluded)", len(result.Repos))
	}
	if result.Repos[0].Name != "shallow" {
		t.Errorf("name = %q, want %q", result.Repos[0].Name, "shallow")
	}
}

func TestScanRepos_EmptyDir(t *testing.T) {
	root := t.TempDir()

	result, err := ScanRepos(root, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Repos) != 0 {
		t.Errorf("repos = %d, want 0", len(result.Repos))
	}
	if result.Summary.Total != 0 {
		t.Errorf("total = %d, want 0", result.Summary.Total)
	}
}

func TestScanRepos_SortedByName(t *testing.T) {
	root := t.TempDir()
	makeRepo(t, root, "zulu", true)
	makeRepo(t, root, "alpha", true)
	makeRepo(t, root, "mike", true)

	result, err := ScanRepos(root, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Repos) != 3 {
		t.Fatalf("repos = %d, want 3", len(result.Repos))
	}
	if result.Repos[0].Name != "alpha" {
		t.Errorf("repos[0] = %q, want %q", result.Repos[0].Name, "alpha")
	}
	if result.Repos[1].Name != "mike" {
		t.Errorf("repos[1] = %q, want %q", result.Repos[1].Name, "mike")
	}
	if result.Repos[2].Name != "zulu" {
		t.Errorf("repos[2] = %q, want %q", result.Repos[2].Name, "zulu")
	}
}

func TestScanRepos_ValidRepoHasSummary(t *testing.T) {
	root := t.TempDir()
	makeRepo(t, root, "goodrepo", true)

	result, err := ScanRepos(root, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Repos) != 1 {
		t.Fatalf("repos = %d, want 1", len(result.Repos))
	}
	repo := result.Repos[0]
	if repo.Summary == nil {
		t.Fatal("summary is nil for valid repo")
	}
	if repo.Summary.Total != 20 {
		t.Errorf("total checks = %d, want 20", repo.Summary.Total)
	}
}
