# Implementation Summary: PM Agent Workflow CLI

> **Status:** Implemented and verified against [AgentAPI OpenAPI spec](https://github.com/coder/agentapi/blob/main/openapi.json)
>
> **Tested:** Full end-to-end workflow producing 5 spec documents + working Go codebase

This document describes the actual implementation. The earlier research explored Claude Agent SDK and other frameworks, but the final implementation uses a simpler CLI-first approach with AgentAPI for faster iteration and reduced complexity.

## Architecture

```
┌─────────────────────────────────────────────────┐
│           pm-agents (CLI orchestrator)           │
│  - Parse PRD, spawn agents, route tasks          │
│  - ~800 lines of Go code                         │
└─────────────────────┬───────────────────────────┘
                      │ HTTP (localhost)
        ┌─────────────┼─────────────┐
        ▼             ▼             ▼
   AgentAPI:3284  AgentAPI:3285  AgentAPI:3286
        │             │             │
        ▼             ▼             ▼
   Claude Code    Claude Code   Claude Code
   (Design)       (Tech)        (QA)
        │             │             │
        └─────────────┴─────────────┘
                      │
               Shared filesystem
            (prd.md, outputs/*.md)
```

## Technology Stack

| Component | Choice | Rationale |
|-----------|--------|-----------|
| Language | Go | Single binary, fast startup, excellent concurrency |
| CLI Framework | Cobra | Industry standard for Go CLIs |
| Agent Control | [AgentAPI](https://github.com/coder/agentapi) | Simple HTTP wrapper around Claude Code |
| Config | YAML | Human-readable, familiar |
| Output | Markdown | Universal, version-controllable |

## Project Structure

```
pm-agent-workflow/
├── cmd/pm-agents/
│   └── main.go              # Entry point
├── internal/
│   ├── agent/
│   │   └── manager.go       # Agent lifecycle management
│   ├── api/
│   │   └── client.go        # AgentAPI HTTP client
│   ├── cmd/
│   │   ├── root.go          # Base command
│   │   ├── run.go           # Main workflow command
│   │   ├── status.go        # Check running agents
│   │   ├── logs.go          # View agent history
│   │   ├── message.go       # Send guidance
│   │   ├── stop.go          # Stop agents
│   │   ├── init.go          # Initialize config
│   │   └── agents.go        # List/show agents
│   └── config/
│       └── config.go        # YAML config loading
├── .github/workflows/
│   ├── build.yml            # CI pipeline
│   └── release.yml          # Release automation
├── docs/                    # Documentation
├── examples/                # Sample PRDs
├── .goreleaser.yml          # Release configuration
├── .golangci.yml            # Linter configuration
├── Makefile                 # Build automation
└── README.md
```

## Key Components

### 1. Agent Manager (`internal/agent/manager.go`)

Handles agent lifecycle:
- **Spawn**: Start AgentAPI process with unique port
- **Health Check**: Wait for `/status` endpoint
- **Task Dispatch**: Send prompt via `POST /message`
- **Monitor**: Poll status until `stable`
- **Cleanup**: Kill process group on completion

### 2. AgentAPI Client (`internal/api/client.go`)

HTTP client for [AgentAPI](https://github.com/coder/agentapi) endpoints:

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/status` | GET | Check agent state (`running`/`stable`) |
| `/message` | POST | Send message to agent |
| `/messages` | GET | Get conversation history |

**Request/Response formats (verified):**

```go
// POST /message request body
type Message struct {
    Content string `json:"content"` // Required
    Type    string `json:"type"`    // "user" or "raw"
}

// GET /status response
type Status struct {
    Status string `json:"status"` // "running" or "stable"
}

// GET /messages response (note: wrapped in object, not direct array)
type MessagesResponse struct {
    Messages []ConversationMessage `json:"messages"`
}

type ConversationMessage struct {
    ID      int    `json:"id"`
    Role    string `json:"role"`    // "user" or "agent"
    Content string `json:"content"`
    Time    string `json:"time"`
}
```

**Important: Agent Startup Behavior**

Claude Code starts in `"running"` state while loading. The CLI must:
1. Wait for API to respond (`WaitForHealthy`)
2. Wait for `"stable"` state (`WaitForStable`)
3. Only then send the initial task message

### 3. Configuration (`internal/config/config.go`)

YAML-based configuration with:
- Default agent prompts (5 agents with clear boundaries)
- Dependency ordering
- Environment variable overrides (`PM_AGENTS_OUTPUT_DIR`, `PM_AGENTS_TIMEOUT`)

**Agent Structure (Refactored):**

| Phase | Agents | Output | Key Principle |
|-------|--------|--------|---------------|
| Specification | architect, qa, security | Markdown docs | Single architect owns all design |
| Implementation | implementer, verifier | Working Go code + tests | Single implementer owns all code |

**Why 5 agents instead of 8:**
- Eliminates overlap between design/tech and backend/database
- Single source of truth for code (implementer)
- Clear contracts between agents

### 4. CLI Commands (`internal/cmd/`)

| Command | Description |
|---------|-------------|
| `run` | Execute workflow on PRD |
| `status` | Check running agents |
| `logs` | View agent conversation |
| `message` | Send guidance to agent |
| `stop` | Stop running agents |
| `init` | Create config file |
| `agents` | List/show agent definitions |
| `version` | Print version |

## Execution Flow

```
1. User runs: pm-agents run ./prd.md

2. CLI loads config (or uses defaults)
   - Reads .pm-agents/config.yaml if exists
   - Applies environment variable overrides

3. For each selected agent:
   a. Allocate port (3284, 3285, ...)
   b. Spawn: agentapi server --port <port> -- claude
   c. Wait for health check (30s timeout)
   d. Send task prompt with PRD path
   e. Monitor status until stable
   f. Verify output file created
   g. Cleanup AgentAPI process

4. Print summary with results
```

## Parallel vs Sequential Mode

**Parallel (default):**
- All agents spawn simultaneously
- Each reads whatever files exist
- Fast but no dependency guarantees

**Sequential (`--sequential`):**
- Topological sort based on `depends_on`
- Each agent waits for dependencies
- Slower but guaranteed order

## State Management

- State file: `/tmp/pm-agents-state.json`
- Contains: `{"agent_name": port_number, ...}`
- Used by `status`, `logs`, `message`, `stop` commands
- Cleared on run completion

## CI/CD Pipeline

### Build Pipeline (`.github/workflows/build.yml`)
1. **Code Quality**: golangci-lint
2. **Security**: govulncheck, Trivy
3. **Tests**: Race detection, coverage
4. **Build**: Multi-platform (darwin/linux, amd64/arm64)

### Release Pipeline (`.github/workflows/release.yml`)
1. Manual trigger with version bump
2. Creates Git tag
3. GoReleaser builds binaries
4. Creates GitHub Release

## What's NOT Implemented (v1)

Per requirements, these are explicitly out of scope:
- Web dashboard
- Database persistence
- Approval gates
- Session resume after crash
- Cost tracking
- Windows support

## Dependencies

- [AgentAPI](https://github.com/coder/agentapi) - HTTP wrapper for Claude Code
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [YAML.v3](https://gopkg.in/yaml.v3) - Configuration parsing

## Development

```bash
# Build
make build

# Run tests
make test

# Lint
make lint

# Release snapshot
make release-snapshot
```

## Verified Workflow

The following end-to-end workflow was tested successfully:

```bash
# Run all 5 agents in sequential mode
pm-agents run ./examples/task-manager-prd.md --sequential -v
```

**Outputs Generated:**

| Agent | Output | Description |
|-------|--------|-------------|
| architect | `architecture.md` | System design, API specs, data models |
| qa | `test-plan.md` | Test cases, acceptance criteria |
| security | `security-assessment.md` | Threat model, mitigations |
| implementer | `code/*` | Complete Go codebase |
| verifier | `verification-report.md`, `code/*_test.go` | Compliance check, tests |

**Generated Code Structure:**
```
outputs/code/
├── cmd/server/main.go
├── internal/
│   ├── handler/       # HTTP handlers
│   ├── service/       # Business logic
│   ├── repository/    # Database ops
│   ├── model/         # Data models
│   ├── middleware/    # Auth, logging
│   ├── config/        # Configuration
│   └── db/            # DB utilities
├── migrations/        # PostgreSQL migrations
├── go.mod, Dockerfile, Makefile
└── sqlc.yaml
```

**Build Verification:**
```bash
cd outputs/code
go mod tidy
go build ./...  # ✓ Passes
go vet ./...    # ✓ Passes
```

## References

- [AgentAPI GitHub](https://github.com/coder/agentapi)
- [AgentAPI HN Discussion](https://news.ycombinator.com/item?id=43719447)
- [Cobra CLI Framework](https://github.com/spf13/cobra)
