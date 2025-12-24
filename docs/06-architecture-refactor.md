# Architecture Refactor: Agent Boundaries & Communication

> **Fact-checked:** Original "overlap" assessment was partially incorrect. See analysis below.

## Analysis of Original 8-Agent Design

### Was There Really Overlap?

**Original claim:** tech/backend/database agents overlapped.

**Fact check:** This was a **spec → implementation flow**, not overlap:
```
tech agent      → "Data models and database schema" (DOCUMENTATION)
backend agent   → models.go, repository/*.go (CODE)
database agent  → migrations/*.sql, db.go (CODE)
```

**The actual issue:** Both `backend` and `database` touch `internal/db/` area, causing potential file conflicts.

### Original 8 Agents

| Agent | Type | Output | Issue |
|-------|------|--------|-------|
| design | spec | design-spec.md | Fine |
| tech | spec | technical-requirements.md | Fine |
| qa | spec | test-plan.md | Fine |
| security | spec | security-assessment.md | Fine |
| infra | spec | infrastructure-plan.md | Fine |
| backend | code | internal/* | Conflicts with database on db.go |
| database | code | migrations/, db.go | Conflicts with backend |
| tests | code | *_test.go | Fine, but depends on both |

## Rationale for 5-Agent Refactor

### 2. No Inter-Agent Communication

```
Current:  PRD → Agent → File → Agent → File
                  ↓           ↓
             (writes)    (reads stale)

Problem: If backend changes API, database doesn't know
```

## Proposed Refactor

### New Agent Structure (5 agents, clear boundaries)

```
┌─────────────────────────────────────────────────────────────┐
│                    SPECIFICATION PHASE                       │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────┐     ┌─────────┐     ┌─────────┐               │
│  │ architect│────▶│   qa    │────▶│ security│               │
│  │         │     │         │     │         │               │
│  └─────────┘     └─────────┘     └─────────┘               │
│       │                                                     │
│       ▼                                                     │
│  Outputs:                                                   │
│  - architecture.md (APIs, data models, component design)    │
│  - test-plan.md                                             │
│  - security-assessment.md                                   │
│                                                             │
├─────────────────────────────────────────────────────────────┤
│                   IMPLEMENTATION PHASE                       │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────────────────────────────────┐               │
│  │              implementer                 │               │
│  │  (single agent owns ALL code)            │               │
│  │                                          │               │
│  │  - cmd/server/main.go                    │               │
│  │  - internal/handler/*.go                 │               │
│  │  - internal/service/*.go                 │               │
│  │  - internal/repository/*.go              │               │
│  │  - internal/model/*.go                   │               │
│  │  - migrations/*.sql                      │               │
│  │  - go.mod, Dockerfile, Makefile          │               │
│  └─────────────────────────────────────────┘               │
│                         │                                   │
│                         ▼                                   │
│  ┌─────────────────────────────────────────┐               │
│  │              verifier                    │               │
│  │  (reviews code, writes tests)            │               │
│  │                                          │               │
│  │  - internal/**/*_test.go                 │               │
│  │  - Validates against specs               │               │
│  │  - Reports discrepancies                 │               │
│  └─────────────────────────────────────────┘               │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Agent Definitions

| Agent | Role | Inputs | Outputs |
|-------|------|--------|---------|
| **architect** | System design (replaces design+tech) | PRD | `architecture.md` (UI, API, data models, infra) |
| **qa** | Test planning | PRD, architecture | `test-plan.md` |
| **security** | Security review | PRD, architecture | `security-assessment.md` |
| **implementer** | ALL code (replaces backend+database) | PRD, architecture, security | `code/*` (complete codebase) |
| **verifier** | Tests + validation (replaces tests) | PRD, all specs, code | `code/*_test.go`, `verification-report.md` |

### Why This Works

1. **No overlap**: One agent owns all code, another owns all tests
2. **Clear handoffs**: Specs → Code → Tests (linear)
3. **Single source of truth**: `implementer` decides models, schema, and API together

## Inter-Agent Communication (v2 Consideration)

### Option A: Iterative Rounds (Recommended for v2)

```
Round 1: architect → qa → security → implementer → verifier
                                           │
                                           ▼
                                    verification-report.md
                                    (lists issues found)
                                           │
Round 2: ◀──────────────────────────────────┘
         architect re-reads report, updates specs
         implementer fixes issues
         verifier re-validates

Repeat until: verifier reports "PASSED" or max_rounds reached
```

Implementation:
```go
type RunConfig struct {
    MaxRounds     int  // default: 3
    AutoIterate   bool // if true, auto-run rounds until pass
}
```

### Option B: Contract Validation (More Complex)

Each agent produces a contract file:
```yaml
# implementer.contract.yaml
exports:
  - type: api
    path: /api/v1/users
    methods: [GET, POST, PUT, DELETE]
  - type: model
    name: User
    fields: [id, email, name, created_at]

imports:
  - from: architect
    expects: [User model definition, /users API spec]
```

Orchestrator validates contracts match before proceeding.

### Option C: Event-Driven (Complex, Not Recommended for v1)

```
┌────────────┐  change_event  ┌────────────┐
│ implementer│───────────────▶│  message   │
└────────────┘                │   queue    │
                              └─────┬──────┘
                                    │
                    ┌───────────────┼───────────────┐
                    ▼               ▼               ▼
              ┌─────────┐    ┌─────────┐    ┌─────────┐
              │verifier │    │ architect│    │   qa    │
              └─────────┘    └─────────┘    └─────────┘
```

Too complex for CLI tool.

## Recommendation

### v1 (Now): Simplified Agents
- 5 agents with clear boundaries
- Sequential execution
- User manually re-runs if needed

### v2 (Future): Iterative Rounds
- Add `--iterate` flag
- Verifier produces structured report
- Auto-rerun until pass or max rounds

## Migration Path

```bash
# v1 behavior (current)
pagent run prd.md --sequential

# v1.1 (with refactored agents)
pagent run prd.md --sequential
# Runs: architect → qa → security → implementer → verifier

# v2 (with iteration)
pagent run prd.md --iterate --max-rounds 3
# Runs multiple rounds until verifier passes
```

---

## Architecture Improvements (Post-Refactor)

The following improvements were made to enhance maintainability and extensibility:

### 1. Dependency-Level Parallelism

**Problem:** Original parallel mode ran all agents simultaneously, ignoring dependencies.

**Solution:** Agents now run in dependency levels:
```
Level 0: architect (no deps)
Level 1: qa, security (parallel, both depend on architect)
Level 2: implementer (depends on architect, security)
Level 3: verifier (depends on implementer, qa)
```

Each level completes before the next starts. Faster than sequential while respecting dependencies.

### 2. Shared Types Package

**Problem:** Type duplication between `config` and `prompt` packages.

**Solution:** Canonical types in `internal/types/types.go`:
- `TechStack` - Technology stack preferences
- `ArchitecturePreferences` - Architectural style options
- Type aliases in `config` and `prompt` packages reference the shared types

### 3. Content-Hash Resume Mode

**Problem:** Simple file mtime comparison for resume was unreliable.

**Solution:** SHA-256 content hashing (`internal/state/resume.go`):
- Hash all input files
- Hash configuration (persona, stack, preferences)
- Hash dependency outputs at generation time
- Detect changes by comparing current hashes to stored hashes

### 4. Orchestrator Interface

**Problem:** Tight coupling between CLI and agent manager made testing difficult.

**Solution:** `Orchestrator` interface (`internal/agent/orchestrator.go`):
```go
type Orchestrator interface {
    RunAgent(ctx context.Context, name string) Result
    TopologicalSort(agents []string) []string
    GetDependencyLevels(agents []string) [][]string
    ExpandWithDependencies(agents []string) []string
    GetTransitiveDependencies(agentName string) []string
    StopAll()
    GetRunningAgents() []*RunningAgent
}
```

Enables:
- Unit testing with mock implementations
- Alternative implementations (e.g., remote agent execution)
- Dependency injection

### 5. Transitive Dependency Resolution

**Problem:** Running a subset of agents (e.g., `--agents verifier`) didn't auto-include dependencies.

**Solution:** `ExpandWithDependencies()` method:
- Takes requested agents
- Expands to include all transitive dependencies
- Returns in dependency order

Example: `--agents verifier` expands to `architect → qa → security → implementer → verifier`

### 6. Configurable Preferences

**Problem:** One-size-fits-all prompts didn't suit different user needs.

**Solution:** Extensive configuration options:

```yaml
persona: balanced  # minimal, balanced, production

preferences:
  stateless: false
  api_style: rest       # rest, graphql, grpc
  language: go          # go, python, typescript, java, rust
  testing_depth: unit   # none, unit, integration, e2e
  documentation_level: standard
  dependency_style: standard
  error_handling: structured
  containerized: true
  include_ci: true
  include_iac: true

stack:
  cloud: aws
  compute: kubernetes
  database: postgres
  cache: redis
  # ... more options
```

Prompts adapt based on these settings.
