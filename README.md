# PM Agent Workflow

[![Build & Verify](https://github.com/tuannvm/pm-agent-workflow/actions/workflows/build.yml/badge.svg)](https://github.com/tuannvm/pm-agent-workflow/actions/workflows/build.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/tuannvm/pm-agent-workflow)](https://goreportcard.com/report/github.com/tuannvm/pm-agent-workflow)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A CLI tool that orchestrates Claude Code specialist agents to transform PRDs into actionable deliverables.

## Overview

`pm-agents` uses the [AgentAPI](https://github.com/coder/agentapi) library to spawn and manage specialist agents that produce:

**Specification Documents:**
- Design specifications
- Technical requirements documents
- Test plans
- Security assessments
- Infrastructure plans

**Working Code (Developer Agents):**
- Go backend API implementation
- PostgreSQL database migrations
- Unit and integration tests

```
┌─────────────────────────────────────────────────┐
│           pm-agents (CLI orchestrator)           │
│  - Parse PRD, spawn agents, route tasks          │
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

## Prerequisites

- [Claude Code](https://claude.ai/claude-code) installed and authenticated
- Go 1.21+ (for building from source)

> **Note:** No external AgentAPI binary required. `pm-agents` uses the AgentAPI library directly.

## Installation

### From Releases

Download the latest release from [GitHub Releases](https://github.com/tuannvm/pm-agent-workflow/releases).

### From Source

```bash
git clone https://github.com/tuannvm/pm-agent-workflow
cd pm-agent-workflow
make build
```

### Install to PATH

```bash
make install
```

## Quick Start

```bash
# Run all 5 agents (specs + code) in dependency order
pm-agents run ./prd.md --sequential -v

# Output files will be in ./outputs/
ls outputs/
# architecture.md          <- System design (architect)
# test-plan.md             <- Test cases (qa)
# security-assessment.md   <- Security review (security)
# verification-report.md   <- Compliance check (verifier)
# code/                    <- Complete codebase (implementer)

# Verify generated code compiles
cd outputs/code && go mod tidy && go build ./...
```

### Specs Only (No Code)

```bash
pm-agents run ./prd.md --agents architect,qa,security --sequential
```

## Usage

### Run Agents

```bash
# Run all agents in parallel (default)
pm-agents run ./prd.md

# Run specific agents only
pm-agents run ./prd.md --agents design,tech

# Run in dependency order (sequential)
pm-agents run ./prd.md --sequential

# Custom output directory
pm-agents run ./prd.md --output ./docs/specs/

# With verbose output
pm-agents run ./prd.md -v
```

### Monitor Agents

```bash
# Check status of running agents
pm-agents status

# View agent conversation history
pm-agents logs design
pm-agents logs tech
```

### Interact with Agents

```bash
# Send guidance to an idle agent
pm-agents message design "Focus more on mobile UX"
pm-agents message tech "Use REST API, not GraphQL"
```

### Stop Agents

```bash
# Stop a specific agent
pm-agents stop tech

# Stop all agents
pm-agents stop --all
```

### Configuration

```bash
# Initialize config in current directory
pm-agents init

# This creates .pm-agents/config.yaml with default prompts
# Edit to customize agent behavior
```

### Other Commands

```bash
# List all agent types
pm-agents agents list

# Show agent prompt template
pm-agents agents show design

# Print version
pm-agents version
```

## Specialist Agents

### Specification Phase

| Agent | Output | Dependencies | Role |
|-------|--------|--------------|------|
| architect | `architecture.md` | - | System design, API, data models |
| qa | `test-plan.md` | architect | Test strategy and cases |
| security | `security-assessment.md` | architect | Threat model, mitigations |

### Implementation Phase

| Agent | Output | Dependencies | Role |
|-------|--------|--------------|------|
| implementer | `code/*` | architect, security | ALL code (API, DB, migrations) |
| verifier | `code/*_test.go` | implementer, qa | Tests + verification report |

**Key Design:** Single `implementer` owns all code to prevent conflicts. Single `verifier` validates against specs.

## Configuration

Create `.pm-agents/config.yaml` to customize:

```yaml
output_dir: ./outputs
timeout: 300  # seconds per agent

# Implementation style (minimal, balanced, production)
persona: balanced

# Architecture preferences
preferences:
  stateless: false          # true = event-driven, false = database-backed
  api_style: rest           # rest, graphql, grpc
  language: go              # go, python, typescript, java, rust
  testing_depth: unit       # none, unit, integration, e2e
  documentation_level: standard  # minimal, standard, comprehensive
  dependency_style: minimal      # minimal (stdlib), standard, batteries
  error_handling: structured     # simple, structured, comprehensive
  containerized: true       # Generate Dockerfile
  include_ci: true          # Generate CI/CD pipelines
  include_iac: true         # Generate Terraform/K8s manifests

# Technology stack
stack:
  cloud: aws
  compute: kubernetes
  database: postgres
  cache: redis
  message_queue: kafka      # Only needed if stateless: true
  iac: terraform
  gitops: argocd
  ci: github-actions
  monitoring: prometheus
  logging: stdout

# Agent customization
agents:
  architect:
    output: architecture.md
    depends_on: []

  qa:
    output: test-plan.md
    depends_on: [architect]

  security:
    output: security-assessment.md
    depends_on: [architect]

  implementer:
    output: code/.complete
    depends_on: [architect, security]

  verifier:
    output: code/.verified
    depends_on: [implementer, qa]
```

### Personas

| Persona | Use Case | Key Characteristics |
|---------|----------|---------------------|
| `minimal` | MVP, prototype | Ship fast, simple code, skip observability |
| `balanced` | Growing product | Essential quality, maintainable code |
| `production` | Enterprise | Comprehensive testing, security, observability |

### Prompt Variables

- `{prd_path}` - Absolute path to the PRD file
- `{output_path}` - Absolute path to the output file

### Environment Variables

- `PM_AGENTS_OUTPUT_DIR` - Override output directory
- `PM_AGENTS_TIMEOUT` - Override timeout (seconds)

## How It Works

1. **Parse PRD** - Reads the PRD file path
2. **Spawn Agents** - Uses AgentAPI library to start Claude Code processes directly (no external binary)
3. **Health Check** - Polls `GET /status` until agent responds (2 min timeout)
4. **Send Task** - `POST /message` with prompt containing PRD path and output path
5. **Monitor** - Polls `/status` until `running` -> `stable` transition
6. **Cleanup** - Gracefully terminates agent processes on completion/interrupt

### Parallel vs Sequential

**Parallel mode (default):**
- Agents run concurrently within dependency levels
- Level 0: `architect` (no dependencies)
- Level 1: `qa`, `security` (both depend only on architect, run in parallel)
- Level 2: `implementer` (depends on architect, security)
- Level 3: `verifier` (depends on implementer, qa)
- Each level must complete before the next starts
- Faster than sequential while respecting dependencies

**Sequential mode (`--sequential`):**
- Agents run in strict dependency order (topological sort)
- Each agent waits for the previous to complete
- Slowest but most predictable

**Resume mode (`--resume`):**
- Skips agents whose outputs are up-to-date
- Detects changes via content hashing (SHA-256):
  - Input files changed?
  - Configuration (persona, stack, preferences) changed?
  - Dependency outputs changed?
- Use `--force` to override and regenerate all

## Development

### Prerequisites

- Go 1.21+
- [golangci-lint](https://golangci-lint.run/) (for linting)
- [goreleaser](https://goreleaser.com/) (for releases)

### Commands

```bash
# Build
make build

# Run tests
make test

# Run linter
make lint

# Format code
make fmt

# Build for all platforms
make build-all

# Create release snapshot
make release-snapshot

# Clean
make clean

# Show all targets
make help
```

### Project Structure

```
pm-agent-workflow/
├── cmd/pm-agents/           # Entry point
├── internal/
│   ├── agent/               # Agent lifecycle management and orchestration
│   │   ├── manager.go       # Core orchestration, RunAgent, state management
│   │   ├── executor.go      # Agent lifecycle (spawn, wait, stop)
│   │   ├── scheduler.go     # Dependency resolution (topological sort)
│   │   ├── orchestrator.go  # Interface abstraction for testability
│   │   └── agentapi_lib.go  # AgentAPI library client integration
│   ├── api/                 # HTTP client for agent status polling
│   ├── cmd/                 # CLI commands
│   ├── config/              # Configuration loading
│   ├── input/               # Input file discovery (single file or directory)
│   ├── postprocess/         # Post-execution actions (diff summary, PR description)
│   ├── prompt/              # Prompt template loading and rendering
│   ├── state/               # Resume state management (content hashing)
│   └── types/               # Shared type definitions (TechStack, Preferences)
├── .github/workflows/       # CI/CD pipelines
├── docs/                    # Documentation
├── examples/                # Sample PRDs and config files
├── .goreleaser.yml          # Release configuration
├── .golangci.yml            # Linter configuration
└── Makefile                 # Build automation
```

## Documentation

- [Research Summary](docs/01-research-summary.md) - Background research on multi-agent systems
- [Framework Comparison](docs/02-framework-comparison.md) - Claude SDK vs alternatives analysis
- [Requirements](docs/03-requirements.md) - Full requirements specification
- [Implementation](docs/04-implementation-plan.md) - Implementation details and architecture
- [User Tutorial](docs/05-tutorial.md) - Step-by-step guide to using pm-agents

## Limitations (v1)

- No web dashboard (CLI only)
- No database persistence (state stored in JSON files)
- No approval gates (agents run autonomously)
- No mid-session resume (crash = restart from beginning)
- macOS/Linux only (no Windows)
- Generated code may require minor fixes (verified to compile with `go build`)

## Troubleshooting

### "timeout waiting for agent"

- Check if Claude Code is authenticated: `claude --version`
- Increase timeout: `pm-agents run prd.md --timeout 600`
- Check agent logs: `pm-agents logs <agent>`

### "port already in use"

Kill existing processes:
```bash
pm-agents stop --all
# Or manually:
lsof -i :3284-3290 | awk 'NR>1 {print $2}' | xargs kill
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run `make lint test`
5. Submit a pull request

## License

MIT

## AgentAPI Reference

This tool uses [AgentAPI](https://github.com/coder/agentapi) endpoints (verified against OpenAPI spec):

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/status` | GET | Returns `{"status": "running"\|"stable"}` |
| `/message` | POST | Send `{"content": "...", "type": "user"}` |
| `/messages` | GET | Get conversation history |

Server flags: `--port` (default 3284), `--type`, `--allowed-hosts`, `--initial-prompt`

## Acknowledgments

- [AgentAPI](https://github.com/coder/agentapi) - Go library for Claude Code process management
- [Claude Code](https://claude.ai/claude-code) - AI coding assistant
- [Cobra](https://github.com/spf13/cobra) - CLI framework
