package skills

// DiffStatus indicates the comparison result for an agent.
type DiffStatus string

const (
	DiffAdded     DiffStatus = "added"
	DiffRemoved   DiffStatus = "removed"
	DiffChanged   DiffStatus = "changed"
	DiffIdentical DiffStatus = "identical"
)

// CountDiff holds a before/after pair for an integer count.
type CountDiff struct {
	A int `json:"a"`
	B int `json:"b"`
}

// TokenDiff holds a before/after pair for token counts.
type TokenDiff struct {
	A int64 `json:"a"`
	B int64 `json:"b"`
}

// AgentDiff is the diff result for a single agent.
type AgentDiff struct {
	Name    string     `json:"name"`
	Status  DiffStatus `json:"status"`
	Skills  CountDiff  `json:"skills"`
	Hooks   CountDiff  `json:"hooks"`
	MCP     CountDiff  `json:"mcp"`
	Tokens  TokenDiff  `json:"tokens"`
	Added   []string   `json:"sources_added,omitempty"`
	Removed []string   `json:"sources_removed,omitempty"`
	Common  []string   `json:"sources_common,omitempty"`
}

// DiffSummary holds aggregate diff counts.
type DiffSummary struct {
	Total     int `json:"total"`
	Added     int `json:"added"`
	Removed   int `json:"removed"`
	Changed   int `json:"changed"`
	Identical int `json:"identical"`
}

// DiffResult is the top-level diff output.
type DiffResult struct {
	PathA   string      `json:"path_a"`
	PathB   string      `json:"path_b"`
	Agents  []AgentDiff `json:"agents"`
	Summary DiffSummary `json:"summary"`
}

// DiffConfigs compares agent configurations between two scan results.
func DiffConfigs(a, b *ScanResult) *DiffResult {
	result := &DiffResult{
		PathA: a.Path,
		PathB: b.Path,
	}

	agentsA := make(map[string]AgentResult)
	for _, agent := range a.Agents {
		agentsA[agent.Name] = agent
	}
	agentsB := make(map[string]AgentResult)
	for _, agent := range b.Agents {
		agentsB[agent.Name] = agent
	}

	seen := make(map[string]bool)
	var names []string
	for _, agent := range a.Agents {
		if !seen[agent.Name] {
			seen[agent.Name] = true
			names = append(names, agent.Name)
		}
	}
	for _, agent := range b.Agents {
		if !seen[agent.Name] {
			seen[agent.Name] = true
			names = append(names, agent.Name)
		}
	}

	for _, name := range names {
		aa, inA := agentsA[name]
		bb, inB := agentsB[name]

		d := AgentDiff{Name: name}

		switch {
		case inA && !inB:
			d.Status = DiffRemoved
			d.Skills = CountDiff{A: aa.Skills}
			d.Hooks = CountDiff{A: aa.Hooks}
			d.MCP = CountDiff{A: aa.MCP}
			d.Tokens = TokenDiff{A: aa.Tokens}
			d.Removed = aa.Sources
		case !inA && inB:
			d.Status = DiffAdded
			d.Skills = CountDiff{B: bb.Skills}
			d.Hooks = CountDiff{B: bb.Hooks}
			d.MCP = CountDiff{B: bb.MCP}
			d.Tokens = TokenDiff{B: bb.Tokens}
			d.Added = bb.Sources
		default:
			d.Skills = CountDiff{A: aa.Skills, B: bb.Skills}
			d.Hooks = CountDiff{A: aa.Hooks, B: bb.Hooks}
			d.MCP = CountDiff{A: aa.MCP, B: bb.MCP}
			d.Tokens = TokenDiff{A: aa.Tokens, B: bb.Tokens}
			d.Added, d.Removed, d.Common = diffSources(aa.Sources, bb.Sources)

			if aa.Skills == bb.Skills && aa.Hooks == bb.Hooks && aa.MCP == bb.MCP &&
				len(d.Added) == 0 && len(d.Removed) == 0 {
				d.Status = DiffIdentical
			} else {
				d.Status = DiffChanged
			}
		}

		result.Agents = append(result.Agents, d)
	}

	for _, d := range result.Agents {
		result.Summary.Total++
		switch d.Status {
		case DiffAdded:
			result.Summary.Added++
		case DiffRemoved:
			result.Summary.Removed++
		case DiffChanged:
			result.Summary.Changed++
		case DiffIdentical:
			result.Summary.Identical++
		}
	}

	return result
}

// diffSources computes the set difference between two source slices.
func diffSources(a, b []string) (added, removed, common []string) {
	setA := make(map[string]bool, len(a))
	for _, s := range a {
		setA[s] = true
	}
	setB := make(map[string]bool, len(b))
	for _, s := range b {
		setB[s] = true
	}

	for _, s := range a {
		if setB[s] {
			common = append(common, s)
		} else {
			removed = append(removed, s)
		}
	}
	for _, s := range b {
		if !setA[s] {
			added = append(added, s)
		}
	}
	return added, removed, common
}
