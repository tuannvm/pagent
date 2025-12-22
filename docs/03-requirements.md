# Requirements: PM Agent Workflow CLI

## Problem Statement

As a Product Manager, I need a simple CLI tool that:
1. Accepts a PRD (Product Requirements Document) as input
2. Spawns specialist agents to produce deliverables in parallel
3. Outputs structured documents (design spec, TRD, test plan, etc.)
4. Allows me to intervene when agents need guidance

**Philosophy:** Simple, intuitive, CLI-first. No bloat.

## Technical Architecture

### Overview

A thin CLI orchestrator built on top of AgentAPI:

```
┌─────────────────────────────────────────────────┐
│           pm-agents (CLI orchestrator)           │
│  - Parse PRD, spawn agents, route tasks          │
│  - ~500-800 lines of code total                  │
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

### Design Principles

1. **Minimal dependencies** — AgentAPI binary + standard library
2. **File-based everything** — PRD in, markdown out, no databases
3. **Parallel by default** — Independent agents run simultaneously
4. **CLI-first** — No web UI required; terminal is the interface
5. **Composable** — Works with pipes, scripts, CI/CD

### Technology Stack

| Component | Choice | Rationale |
|-----------|--------|-----------|
| Language | Go | Single binary, fast startup, excellent concurrency |
| Agent Control | AgentAPI | Simple HTTP API, proven, 1k+ stars |
| Agent Runtime | Claude Code | Full-featured, hooks support |
| Config | YAML or flags | Simple, no framework |
| Output | Markdown files | Universal, version-controllable |

**Alternative:** TypeScript/Node.js if Go isn't preferred.

## CLI Interface

### Primary Commands

```bash
# Run all specialists on a PRD
pm-agents run ./prd.md

# Run specific specialists only
pm-agents run ./prd.md --agents design,tech

# Run with custom output directory
pm-agents run ./prd.md --output ./docs/

# Run with dependency ordering (tech waits for design)
pm-agents run ./prd.md --sequential
```

### Agent Interaction

```bash
# Check status of running agents
pm-agents status

# View live output from an agent
pm-agents logs tech --follow

# Send a message to a specific agent (when idle)
pm-agents message design "Focus more on mobile UX"

# Stop a specific agent
pm-agents stop tech

# Stop all agents
pm-agents stop --all
```

### Configuration

```bash
# Initialize config in current directory
pm-agents init

# List available agent types
pm-agents agents list

# Show agent prompt template
pm-agents agents show design
```

## Functional Requirements

### FR-1: PRD Input

- **FR-1.1**: Accept PRD as markdown file path
- **FR-1.2**: Validate file exists and is readable
- **FR-1.3**: Pass PRD path to agents (agents read file directly)

### FR-2: Agent Lifecycle

- **FR-2.1**: Spawn AgentAPI process per specialist
  - Assign unique port (3284, 3285, 3286...)
  - Wait for health check before sending tasks
- **FR-2.2**: Send initial task prompt to each agent
  - Include PRD path and output file path
  - Agent-specific system prompt from config
- **FR-2.3**: Monitor agent status via polling
  - `GET /status` returns `"running"` or `"stable"`
- **FR-2.4**: Cleanup on completion or interrupt
  - Kill AgentAPI processes on SIGINT/SIGTERM
  - Report final status

### FR-3: Specialist Agents

Five specialists, each with a focused prompt:

| Agent | Input | Output | Waits For |
|-------|-------|--------|-----------|
| **design** | PRD | `design-spec.md` | — |
| **tech** | PRD, design-spec | `technical-requirements.md` | design (optional) |
| **qa** | PRD, design-spec, TRD | `test-plan.md` | tech (optional) |
| **security** | PRD, TRD | `security-assessment.md` | tech (optional) |
| **infra** | PRD, TRD | `infrastructure-plan.md` | tech (optional) |

**Default mode:** All run in parallel (agents read whatever files exist).
**Sequential mode:** `--sequential` flag enforces dependency order.

### FR-4: Output Management

- **FR-4.1**: Each agent writes to designated output file
- **FR-4.2**: Default output directory: `./outputs/`
- **FR-4.3**: Overwrite existing files (no versioning in v1)
- **FR-4.4**: Print summary on completion with file paths

### FR-5: Agent Interaction

- **FR-5.1**: Send message to idle agent
  - `POST /message` with `type: "user"`
  - Only works when agent status is `"stable"`
  - CLI waits for stable state or times out
- **FR-5.2**: View agent conversation
  - `GET /messages` returns history
  - CLI formats and displays
- **FR-5.3**: Stream agent output (optional)
  - `GET /events` SSE stream
  - CLI prints updates in real-time

### FR-6: Configuration

- **FR-6.1**: Config file at `.pm-agents/config.yaml`
- **FR-6.2**: Configurable per agent:
  - System prompt (or path to prompt file)
  - Output filename
  - Dependencies (which agents to wait for)
  - Model override (if supported)
- **FR-6.3**: Environment variable overrides
  - `PM_AGENTS_OUTPUT_DIR`
  - `PM_AGENTS_TIMEOUT`

### FR-7: Error Handling

- **FR-7.1**: Detect agent spawn failure
  - AgentAPI process exits unexpectedly
  - Health check timeout (default 30s)
- **FR-7.2**: Report errors clearly
  - Which agent failed
  - Last known status
  - Stderr output if available
- **FR-7.3**: Graceful degradation
  - If one agent fails, others continue
  - Final summary shows partial results

## Non-Functional Requirements

### NFR-1: Performance

- **NFR-1.1**: CLI startup < 100ms (before spawning agents)
- **NFR-1.2**: Agent spawn < 5s per agent
- **NFR-1.3**: Support 5 concurrent agents on typical hardware

### NFR-2: Usability

- **NFR-2.1**: Zero configuration for basic use
  - `pm-agents run prd.md` just works
- **NFR-2.2**: Clear, actionable error messages
- **NFR-2.3**: Progress indication during long operations
- **NFR-2.4**: `--help` for all commands
- **NFR-2.5**: `--verbose` and `--quiet` flags

### NFR-3: Reliability

- **NFR-3.1**: Clean shutdown on Ctrl+C
- **NFR-3.2**: No orphan processes left behind
- **NFR-3.3**: Idempotent operations (re-run safely)

### NFR-4: Portability

- **NFR-4.1**: Works on macOS and Linux
- **NFR-4.2**: Single binary distribution (if Go)
- **NFR-4.3**: No runtime dependencies beyond AgentAPI + Claude Code

## User Stories

### Epic 1: Basic Workflow

**US-1.1**: As a PM, I want to run all specialists on my PRD with a single command.
```bash
pm-agents run ./prd.md
# Spawns 5 agents, waits for completion, outputs files
```

**US-1.2**: As a PM, I want to see progress while agents work.
```bash
pm-agents run ./prd.md
# Output:
# ✓ design: running...
# ✓ tech: running...
# ✓ qa: running...
# ✓ design: completed → outputs/design-spec.md
# ✓ tech: completed → outputs/technical-requirements.md
# ...
```

**US-1.3**: As a PM, I want to run only specific specialists.
```bash
pm-agents run ./prd.md --agents design,tech
```

### Epic 2: Agent Interaction

**US-2.1**: As a PM, I want to check if agents are still working.
```bash
pm-agents status
# Output:
# design: stable (idle)
# tech: running
# qa: running
```

**US-2.2**: As a PM, I want to send guidance to an agent that went off track.
```bash
pm-agents message tech "Focus on REST API, not GraphQL"
# Waits for agent to be idle, sends message, confirms
```

**US-2.3**: As a PM, I want to see what an agent has done so far.
```bash
pm-agents logs design
# Shows conversation history
```

### Epic 3: Configuration

**US-3.1**: As a PM, I want to customize agent prompts for my domain.
```bash
pm-agents init
# Creates .pm-agents/config.yaml with defaults
# Edit prompts as needed
```

**US-3.2**: As a PM, I want to change output directory.
```bash
pm-agents run ./prd.md --output ./docs/specs/
```

### Epic 4: Error Recovery

**US-4.1**: As a PM, I want to stop everything if I made a mistake.
```bash
pm-agents stop --all
# Kills all agents, confirms cleanup
```

**US-4.2**: As a PM, I want to know why an agent failed.
```bash
pm-agents run ./prd.md
# Output:
# ✗ security: failed (timeout waiting for stable state)
# ✓ design: completed
# ...
# Partial results saved. 4/5 agents succeeded.
```

## Configuration File Format

```yaml
# .pm-agents/config.yaml

output_dir: ./outputs
timeout: 300  # seconds per agent

agents:
  design:
    prompt: |
      You are a Design Lead. Read the PRD at {prd_path} and create a design specification.
      Write your output to {output_path}.

      Include:
      - UI/UX requirements
      - User flows (mermaid diagrams)
      - Component specifications
      - Accessibility requirements
    output: design-spec.md
    depends_on: []

  tech:
    prompt: |
      You are a Tech Lead. Read the PRD at {prd_path} and any existing design docs.
      Write your output to {output_path}.

      Include:
      - Architecture decisions
      - API specifications
      - Data models
      - Technical constraints
    output: technical-requirements.md
    depends_on: [design]  # Optional: wait for design in sequential mode

  qa:
    prompt: |
      You are a QA Lead. Read the PRD and existing specs.
      Write your output to {output_path}.

      Include:
      - Test strategy
      - Test cases
      - Acceptance criteria
    output: test-plan.md
    depends_on: [tech]

  security:
    prompt: |
      You are a Security Reviewer. Assess security implications.
      Write your output to {output_path}.

      Include:
      - Threat model
      - Security requirements
      - Risk mitigations
    output: security-assessment.md
    depends_on: [tech]

  infra:
    prompt: |
      You are an Infrastructure Lead. Plan infrastructure needs.
      Write your output to {output_path}.

      Include:
      - Resource requirements
      - Deployment strategy
      - Scaling considerations
    output: infrastructure-plan.md
    depends_on: [tech]
```

## Constraints

### Technical Constraints

- AgentAPI binary must be installed and in PATH
- Claude Code must be installed and authenticated
- Ports 3284-3290 available for agent processes
- macOS or Linux (Windows not supported in v1)

### Scope Constraints (v1)

- No web dashboard (CLI only)
- No persistent state between runs
- No approval gates (agents run autonomously)
- No real-time collaboration
- No cost tracking

## Out of Scope (v1)

- Web dashboard / UI
- Database persistence
- Approval workflow (hooks-based gating)
- Session resume after crash
- Multi-user support
- Windows support
- Cost tracking / token counting
- Custom agent types beyond the five specialists
- Integration with external tools (Jira, Linear, etc.)

## Future Considerations (v2+)

If v1 proves useful, consider:

1. **Approval gates** — Claude Code hooks + HTTP callback
2. **Web dashboard** — Optional, for those who want it
3. **Session persistence** — Resume interrupted runs
4. **Watch mode** — Re-run on PRD file changes
5. **Templates** — Pre-built prompt sets for different domains
6. **Plugins** — Custom agents via config

## Success Metrics

1. **Time to first run**: < 2 minutes from install to first output
2. **Command simplicity**: Core workflow in single command
3. **Reliability**: 95% of runs complete without manual intervention
4. **Output quality**: Generated docs require minimal editing

## Glossary

| Term | Definition |
|------|------------|
| PRD | Product Requirements Document — input specification |
| AgentAPI | HTTP wrapper around Claude Code (coder/agentapi) |
| Specialist | One of 5 agent types (design, tech, qa, security, infra) |
| Stable | AgentAPI status indicating agent is idle and can receive messages |
| Running | AgentAPI status indicating agent is actively processing |

## Implementation Estimate

| Component | Effort | Notes |
|-----------|--------|-------|
| CLI framework + commands | 1-2 days | Cobra (Go) or Commander (TS) |
| Agent spawner | 1 day | Process management, health checks |
| Task router | 1 day | HTTP calls to AgentAPI |
| Config loader | 0.5 day | YAML parsing, defaults |
| Output handling | 0.5 day | File writing, summary |
| Error handling | 1 day | Edge cases, cleanup |
| Testing | 1-2 days | Unit + integration |
| **Total** | **6-8 days** | For MVP |

## Appendix: AgentAPI Reference

> **Verified against:** [coder/agentapi](https://github.com/coder/agentapi) OpenAPI specification

### Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/status` | GET | Check if agent is `"running"` or `"stable"` |
| `/messages` | GET | Get conversation history |
| `/message` | POST | Send message (`type: "user"` or `"raw"`) |
| `/events` | GET | SSE stream of updates |
| `/openapi.json` | GET | OpenAPI specification |
| `/docs` | GET | Documentation UI |
| `/chat` | GET | Web chat interface |

### Server Command

```bash
agentapi server --port 3284 -- claude
```

**Available flags:**
| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--port` | `-p` | `3284` | Server port |
| `--type` | `-t` | (auto) | Override agent type |
| `--allowed-hosts` | `-a` | `localhost` | Permitted hostnames |
| `--allowed-origins` | `-o` | `localhost:3284,3000,3001` | CORS origins |
| `--initial-prompt` | `-I` | (none) | Starting prompt |

### POST /message Request

```bash
curl -X POST localhost:3284/message \
  -H "Content-Type: application/json" \
  -d '{"content": "Read prd.md and create design spec", "type": "user"}'
```

**Request body:**
```json
{
  "content": "string (required) - message text",
  "type": "string (required) - 'user' or 'raw'"
}
```

- `type: "user"` - Message is logged and submitted to agent
- `type: "raw"` - Message written directly to terminal

### GET /status Response

```json
{
  "status": "stable|running",
  "agent_type": "claude"
}
```

- `stable` - Agent is idle, ready for input
- `running` - Agent is processing a message
