package skills

import "testing"

func TestDiffConfigs_Identical(t *testing.T) {
	a := &ScanResult{
		Path: "/path/a",
		Agents: []AgentResult{
			{Name: AgentClaudeCode, Skills: 3, Hooks: 1, MCP: 0, Tokens: 100, Sources: []string{"CLAUDE.md", ".claude/skills/"}},
			{Name: AgentCline, Skills: 2, Hooks: 0, MCP: 0, Tokens: 50, Sources: []string{".clinerules/"}},
		},
	}
	b := &ScanResult{
		Path: "/path/b",
		Agents: []AgentResult{
			{Name: AgentClaudeCode, Skills: 3, Hooks: 1, MCP: 0, Tokens: 100, Sources: []string{"CLAUDE.md", ".claude/skills/"}},
			{Name: AgentCline, Skills: 2, Hooks: 0, MCP: 0, Tokens: 50, Sources: []string{".clinerules/"}},
		},
	}

	result := DiffConfigs(a, b)
	if result.Summary.Total != 2 {
		t.Errorf("total = %d, want 2", result.Summary.Total)
	}
	if result.Summary.Identical != 2 {
		t.Errorf("identical = %d, want 2", result.Summary.Identical)
	}
	for _, d := range result.Agents {
		if d.Status != DiffIdentical {
			t.Errorf("agent %s: status = %s, want identical", d.Name, d.Status)
		}
	}
}

func TestDiffConfigs_AgentOnlyInA(t *testing.T) {
	a := &ScanResult{
		Path: "/path/a",
		Agents: []AgentResult{
			{Name: AgentCline, Skills: 2, Sources: []string{".clinerules/"}},
		},
	}
	b := &ScanResult{Path: "/path/b"}

	result := DiffConfigs(a, b)
	if result.Summary.Removed != 1 {
		t.Errorf("removed = %d, want 1", result.Summary.Removed)
	}
	if result.Agents[0].Status != DiffRemoved {
		t.Errorf("status = %s, want removed", result.Agents[0].Status)
	}
	if len(result.Agents[0].Removed) != 1 || result.Agents[0].Removed[0] != ".clinerules/" {
		t.Errorf("removed sources = %v, want [.clinerules/]", result.Agents[0].Removed)
	}
}

func TestDiffConfigs_AgentOnlyInB(t *testing.T) {
	a := &ScanResult{Path: "/path/a"}
	b := &ScanResult{
		Path: "/path/b",
		Agents: []AgentResult{
			{Name: AgentCursor, Skills: 1, MCP: 2, Sources: []string{".cursor/rules/", "~/.cursor/mcp.json"}},
		},
	}

	result := DiffConfigs(a, b)
	if result.Summary.Added != 1 {
		t.Errorf("added = %d, want 1", result.Summary.Added)
	}
	if result.Agents[0].Status != DiffAdded {
		t.Errorf("status = %s, want added", result.Agents[0].Status)
	}
	if len(result.Agents[0].Added) != 2 {
		t.Errorf("added sources = %d, want 2", len(result.Agents[0].Added))
	}
}

func TestDiffConfigs_CountsChanged(t *testing.T) {
	a := &ScanResult{
		Path: "/path/a",
		Agents: []AgentResult{
			{Name: AgentClaudeCode, Skills: 3, Hooks: 1, MCP: 0, Sources: []string{"CLAUDE.md"}},
		},
	}
	b := &ScanResult{
		Path: "/path/b",
		Agents: []AgentResult{
			{Name: AgentClaudeCode, Skills: 5, Hooks: 1, MCP: 1, Sources: []string{"CLAUDE.md"}},
		},
	}

	result := DiffConfigs(a, b)
	if result.Summary.Changed != 1 {
		t.Errorf("changed = %d, want 1", result.Summary.Changed)
	}
	d := result.Agents[0]
	if d.Skills.A != 3 || d.Skills.B != 5 {
		t.Errorf("skills = %d -> %d, want 3 -> 5", d.Skills.A, d.Skills.B)
	}
	if d.MCP.A != 0 || d.MCP.B != 1 {
		t.Errorf("mcp = %d -> %d, want 0 -> 1", d.MCP.A, d.MCP.B)
	}
}

func TestDiffConfigs_SourcesChanged(t *testing.T) {
	a := &ScanResult{
		Path: "/path/a",
		Agents: []AgentResult{
			{Name: AgentClaudeCode, Skills: 3, Sources: []string{"CLAUDE.md", ".claude/skills/"}},
		},
	}
	b := &ScanResult{
		Path: "/path/b",
		Agents: []AgentResult{
			{Name: AgentClaudeCode, Skills: 3, Sources: []string{"CLAUDE.md", ".claude/settings.local.json"}},
		},
	}

	result := DiffConfigs(a, b)
	if result.Summary.Changed != 1 {
		t.Errorf("changed = %d, want 1", result.Summary.Changed)
	}
	d := result.Agents[0]
	if len(d.Added) != 1 || d.Added[0] != ".claude/settings.local.json" {
		t.Errorf("added = %v, want [.claude/settings.local.json]", d.Added)
	}
	if len(d.Removed) != 1 || d.Removed[0] != ".claude/skills/" {
		t.Errorf("removed = %v, want [.claude/skills/]", d.Removed)
	}
	if len(d.Common) != 1 || d.Common[0] != "CLAUDE.md" {
		t.Errorf("common = %v, want [CLAUDE.md]", d.Common)
	}
}

func TestDiffConfigs_BothEmpty(t *testing.T) {
	a := &ScanResult{Path: "/path/a"}
	b := &ScanResult{Path: "/path/b"}

	result := DiffConfigs(a, b)
	if len(result.Agents) != 0 {
		t.Errorf("agents = %d, want 0", len(result.Agents))
	}
	if result.Summary.Total != 0 {
		t.Errorf("total = %d, want 0", result.Summary.Total)
	}
}

func TestDiffConfigs_TokensDifferOnly(t *testing.T) {
	a := &ScanResult{
		Path: "/path/a",
		Agents: []AgentResult{
			{Name: AgentCline, Skills: 2, Tokens: 100, Sources: []string{".clinerules/"}},
		},
	}
	b := &ScanResult{
		Path: "/path/b",
		Agents: []AgentResult{
			{Name: AgentCline, Skills: 2, Tokens: 200, Sources: []string{".clinerules/"}},
		},
	}

	result := DiffConfigs(a, b)
	if result.Agents[0].Status != DiffIdentical {
		t.Errorf("status = %s, want identical (tokens are informational)", result.Agents[0].Status)
	}
	if result.Agents[0].Tokens.A != 100 || result.Agents[0].Tokens.B != 200 {
		t.Errorf("tokens not preserved: %d -> %d", result.Agents[0].Tokens.A, result.Agents[0].Tokens.B)
	}
}

func TestDiffConfigs_OrderPreserved(t *testing.T) {
	a := &ScanResult{
		Path: "/path/a",
		Agents: []AgentResult{
			{Name: AgentClaudeCode, Skills: 1, Sources: []string{"CLAUDE.md"}},
			{Name: AgentCline, Skills: 1, Sources: []string{".clinerules/"}},
		},
	}
	b := &ScanResult{
		Path: "/path/b",
		Agents: []AgentResult{
			{Name: AgentCursor, Skills: 1, Sources: []string{".cursor/rules/"}},
			{Name: AgentCline, Skills: 1, Sources: []string{".clinerules/"}},
		},
	}

	result := DiffConfigs(a, b)
	if len(result.Agents) != 3 {
		t.Fatalf("agents = %d, want 3", len(result.Agents))
	}
	// A's agents first (claude-code, cline), then B-only (cursor).
	if result.Agents[0].Name != AgentClaudeCode {
		t.Errorf("agents[0] = %s, want claude-code", result.Agents[0].Name)
	}
	if result.Agents[1].Name != AgentCline {
		t.Errorf("agents[1] = %s, want cline", result.Agents[1].Name)
	}
	if result.Agents[2].Name != AgentCursor {
		t.Errorf("agents[2] = %s, want cursor", result.Agents[2].Name)
	}
}

// --- diffSources ---

func TestDiffSources_SetDifference(t *testing.T) {
	a := []string{"CLAUDE.md", ".claude/skills/", "~/.claude/settings.json"}
	b := []string{"CLAUDE.md", ".claude/settings.local.json", "~/.claude/settings.json"}

	added, removed, common := diffSources(a, b)

	if len(added) != 1 || added[0] != ".claude/settings.local.json" {
		t.Errorf("added = %v, want [.claude/settings.local.json]", added)
	}
	if len(removed) != 1 || removed[0] != ".claude/skills/" {
		t.Errorf("removed = %v, want [.claude/skills/]", removed)
	}
	if len(common) != 2 {
		t.Errorf("common = %d, want 2", len(common))
	}
}

func TestDiffSources_Empty(t *testing.T) {
	added, removed, common := diffSources(nil, nil)
	if added != nil || removed != nil || common != nil {
		t.Errorf("expected all nil, got added=%v removed=%v common=%v", added, removed, common)
	}
}

func TestDiffSources_NoOverlap(t *testing.T) {
	a := []string{"CLAUDE.md"}
	b := []string{".cursor/rules/"}

	added, removed, common := diffSources(a, b)
	if len(added) != 1 {
		t.Errorf("added = %d, want 1", len(added))
	}
	if len(removed) != 1 {
		t.Errorf("removed = %d, want 1", len(removed))
	}
	if common != nil {
		t.Errorf("common = %v, want nil", common)
	}
}
