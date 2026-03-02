package validator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ScanStatus values for repo-level results.
const (
	ScanStatusPass    = "pass"
	ScanStatusFail    = "fail"
	ScanStatusPartial = "partial"
	ScanStatusMissing = "missing"
)

// RepoResult holds the validation outcome for a single repo.
type RepoResult struct {
	Name    string   `json:"name"`
	Path    string   `json:"path"`
	Status  string   `json:"status"`
	Summary *Summary `json:"summary,omitempty"`
}

// ScanSummary holds aggregate counts across all repos.
type ScanSummary struct {
	Total   int `json:"total"`
	Pass    int `json:"pass"`
	Fail    int `json:"fail"`
	Partial int `json:"partial"`
	Missing int `json:"missing"`
}

// ScanResult holds the batch validation outcome.
type ScanResult struct {
	Path    string       `json:"path"`
	Repos   []RepoResult `json:"repos"`
	Summary ScanSummary  `json:"summary"`
}

// skipDirs are directory names to skip during scanning.
var skipDirs = map[string]bool{
	"node_modules": true,
	"vendor":       true,
	"__pycache__":  true,
}

// ScanRepos walks root up to depth levels deep, finds git repos, and
// validates each against the ANCC convention.
func ScanRepos(root string, depth int) (*ScanResult, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	result := &ScanResult{Path: root}

	repos, err := findRepos(root, depth)
	if err != nil {
		return nil, fmt.Errorf("scanning directory: %w", err)
	}

	for _, repoPath := range repos {
		rr := validateRepo(repoPath)
		result.Repos = append(result.Repos, rr)
	}

	sort.Slice(result.Repos, func(i, j int) bool {
		return result.Repos[i].Name < result.Repos[j].Name
	})

	computeScanSummary(result)
	return result, nil
}

// findRepos walks root up to maxDepth levels, returning paths that contain .git/.
func findRepos(root string, maxDepth int) ([]string, error) {
	var repos []string
	err := walkDirs(root, 0, maxDepth, func(dir string) bool {
		gitDir := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
			repos = append(repos, dir)
			return false // don't descend into repos
		}
		return true // keep descending
	})
	return repos, err
}

// walkDirs recursively walks directories up to maxDepth. The visitor returns
// true to continue descending into subdirectories, false to stop.
func walkDirs(dir string, currentDepth, maxDepth int, visit func(string) bool) error {
	if currentDepth > maxDepth {
		return nil
	}

	if !visit(dir) {
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil // skip unreadable directories
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") || skipDirs[name] {
			continue
		}
		if err := walkDirs(filepath.Join(dir, name), currentDepth+1, maxDepth, visit); err != nil {
			return err
		}
	}
	return nil
}

// validateRepo runs Validate on a single repo path and maps the result.
func validateRepo(repoPath string) RepoResult {
	name := filepath.Base(repoPath)
	rr := RepoResult{Name: name, Path: repoPath}

	vr, err := Validate(repoPath)
	if err != nil {
		rr.Status = ScanStatusFail
		return rr
	}

	// Check if SKILL.md exists — if not, mark as missing.
	for _, c := range vr.Checks {
		if c.Name == CheckSkillMDExists && c.Status == StatusFail {
			rr.Status = ScanStatusMissing
			return rr
		}
	}

	rr.Summary = &vr.Summary

	switch vr.Status {
	case OverallPass:
		rr.Status = ScanStatusPass
	case OverallFail:
		rr.Status = ScanStatusFail
	case OverallPartial:
		rr.Status = ScanStatusPartial
	default:
		rr.Status = ScanStatusFail
	}

	return rr
}

func computeScanSummary(r *ScanResult) {
	for _, repo := range r.Repos {
		r.Summary.Total++
		switch repo.Status {
		case ScanStatusPass:
			r.Summary.Pass++
		case ScanStatusFail:
			r.Summary.Fail++
		case ScanStatusPartial:
			r.Summary.Partial++
		case ScanStatusMissing:
			r.Summary.Missing++
		}
	}
}
