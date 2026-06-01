# ANCC Conventions

Rules for building tools that evolve safely under agent modification.

ANCC has six tool requirements and nine governed conventions. Requirements define
what makes a CLI agent-native. Conventions define how those tools stay bounded,
observable, and governable as agents extend them.

## Governed Surfaces

| Surface | What It Governs |
|---------|-----------------|
| Behavior | What the tool does, refuses, emits, and hands off |
| Execution | What agents and tools are allowed to act on |
| Spend | How work is bounded by cost, scope, and context pressure |
| Time | How changes, releases, and compatibility are declared |

## Convention Index

| # | Convention | Primary Surface |
|---|------------|-----------------|
| 1 | scope-boundaries | Behavior |
| 2 | extend-vs-new-tool | Spend |
| 3 | handoff-contracts | Behavior |
| 4 | doctor-output-schema | Behavior |
| 5 | provenance-classification | Behavior |
| 6 | deprecation-and-pruning | Time |
| 7 | temporal-contracts | Time |
| 8 | active-defense | Execution |
| 9 | enforcement-provenance | Execution |

## 9. Enforcement Provenance

What: A standard convention for claims that a tool, agent, policy profile, or
configuration enforces a guardrail.

Why: Enforcement claims are easy to overread. A vendor document or config name can
say "secure mode" while runtime behavior remains advisory. Agents need a
deterministic vocabulary that separates verified blocking, verified advisory
signals, and unverified assertions.

Rule: Any enforcement claim MUST use one of three states:

- `enforcing`: a cited live probe demonstrates that the named guard structurally
  blocks the covered action at runtime.
- `advisory`: a cited live probe demonstrates that the named guard reports,
  configures, or recommends a boundary but does not structurally block the
  covered action.
- `unverified`: no cited live probe is attached. This is the default state.

Evidence requirement: `enforcing` and `advisory` both require cited live probe
evidence. Vendor docs, product names, and local config labels are not enough.
The evidence should name the guard, the action tested, the observed result, and
the scope where the result applies.

Default: Missing or unsupported posture is `unverified`.

Mirror principle: ancc reports enforcement posture and evidence. It does not
decide whether advisory or unverified posture blocks a workflow. CI, governance
tooling, or the operator decides what to do with the reported posture.

Relationship to provenance-classification: provenance-classification describes
where a JSON field came from, such as observed, declared, inferred, or unknown.
enforcement-provenance describes what a guardrail claim means after live probing.
The two conventions compose: an enforcement posture field can still carry ordinary
field provenance.

Worked example: agy exposes policy-profile and secure-mode style configuration.
A live probe found that the project policy was advisory for file reads: macOS TCC
blocked a narrow set of user folders, while credential directories remained
reachable. The correct posture is `advisory`, with that probe attached as
evidence. Without the probe, the posture is `unverified`, not `enforcing`.

Validator: pending. The convention is vocabulary and reporting contract first;
validator enforcement belongs in a separate implementation change.
