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
pm-agents run prd.md --sequential

# v1.1 (with refactored agents)
pm-agents run prd.md --sequential
# Runs: architect → qa → security → implementer → verifier

# v2 (with iteration)
pm-agents run prd.md --iterate --max-rounds 3
# Runs multiple rounds until verifier passes
```
