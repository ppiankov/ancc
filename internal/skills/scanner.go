package skills

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ppiankov/ancc/internal/skillmd"
)

// Scan scans the given directory and user home for agent configurations.
func Scan(projectDir string) (*ScanResult, error) {
	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = ""
	}

	return ScanWithHome(absDir, homeDir)
}

// ScanWithHome is the testable core of Scan with an explicit home directory.
func ScanWithHome(projectDir, homeDir string) (*ScanResult, error) {
	result := &ScanResult{Path: projectDir}

	scanners := []func(string, string) AgentResult{
		scanClaudeCode,
		scanCline,
		scanCursor,
		scanOpenCode,
		scanCodex,
		scanQwen,
	}

	for _, scan := range scanners {
		agent := scan(projectDir, homeDir)
		if agent.Skills > 0 || agent.Hooks > 0 || agent.MCP > 0 {
			result.Agents = append(result.Agents, agent)
		}
	}

	result.Product = findANCCProduct(projectDir)

	return result, nil
}

// findANCCProduct checks for a consumer-facing SKILL.md in the project.
func findANCCProduct(projectDir string) *ANCCProduct {
	paths := []string{
		filepath.Join(projectDir, "docs", "SKILL.md"),
		filepath.Join(projectDir, "SKILL.md"),
	}

	for _, p := range paths {
		sf, err := skillmd.ParseFile(p)
		if err != nil {
			continue
		}
		return &ANCCProduct{
			Path: p,
			Name: sf.Name,
		}
	}
	return nil
}
