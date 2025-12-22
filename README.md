# PM Agent Workflow

[![Build & Verify](https://github.com/tuannvm/pm-agent-workflow/actions/workflows/build.yml/badge.svg)](https://github.com/tuannvm/pm-agent-workflow/actions/workflows/build.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/tuannvm/pm-agent-workflow)](https://goreportcard.com/report/github.com/tuannvm/pm-agent-workflow)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A CLI tool that orchestrates Claude Code specialist agents to transform PRDs into actionable deliverables.

## Overview

`pm-agents` spawns specialist agents via [AgentAPI](https://github.com/coder/agentapi) to produce:

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

- [AgentAPI](https://github.com/coder/agentapi) installed and in PATH
- [Claude Code](https://claude.ai/claude-code) installed and authenticated
- Go 1.21+ (for building from source)

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
# Run all specialists on a PRD (documentation only)
pm-agents run ./prd.md --agents design,tech,qa,security,infra

# Output files will be in ./outputs/
ls outputs/
# design-spec.md
# technical-requirements.md
# test-plan.md
# security-assessment.md
# infrastructure-plan.md
```

### Full PRD-to-Code Pipeline

```bash
# Run all 8 agents (docs + code generation)
pm-agents run ./prd.md --sequential -v

# Generated outputs:
ls outputs/
# Docs: design-spec.md, technical-requirements.md, etc.
# Code: code/cmd/server/main.go, code/internal/*, code/migrations/*

# Verify generated code compiles
cd outputs/code && go mod tidy && go build ./...
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

### Spec Agents (Documentation)

| Agent | Output File | Dependencies |
|-------|-------------|--------------|
| design | `design-spec.md` | - |
| tech | `technical-requirements.md` | design |
| qa | `test-plan.md` | tech |
| security | `security-assessment.md` | tech |
| infra | `infrastructure-plan.md` | tech |

### Developer Agents (Code Generation)

| Agent | Output | Dependencies |
|-------|--------|--------------|
| backend | `code/` (Go API) | tech, security |
| database | `code/migrations/` (SQL) | tech |
| tests | `code/*_test.go` | backend, database, qa |

## Configuration

Create `.pm-agents/config.yaml` to customize:

```yaml
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
    output: design-spec.md
    depends_on: []

  tech:
    prompt: |
      You are a Tech Lead. Create technical requirements.
      Write your output to {output_path}.
    output: technical-requirements.md
    depends_on: [design]
```

### Prompt Variables

- `{prd_path}` - Absolute path to the PRD file
- `{output_path}` - Absolute path to the output file

### Environment Variables

- `PM_AGENTS_OUTPUT_DIR` - Override output directory
- `PM_AGENTS_TIMEOUT` - Override timeout (seconds)

## How It Works

1. **Parse PRD** - Reads the PRD file path
2. **Spawn Agents** - Starts AgentAPI process per specialist (`agentapi server --port <port> -- claude`)
3. **Health Check** - Polls `GET /status` until agent responds (30s timeout)
4. **Send Task** - `POST /message` with prompt containing PRD path and output path
5. **Monitor** - Polls `/status` until `running` -> `stable` transition
6. **Cleanup** - Kills AgentAPI process groups on completion/interrupt

### Parallel vs Sequential

**Parallel mode (default):**
- All agents start simultaneously
- Agents read whatever files exist at runtime
- Faster but dependencies may not be available

**Sequential mode (`--sequential`):**
- Agents run in dependency order (topological sort)
- Each agent waits for dependencies to complete
- Slower but dependencies are guaranteed

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
│   ├── agent/               # Agent lifecycle management
│   ├── api/                 # AgentAPI HTTP client
│   ├── cmd/                 # CLI commands
│   └── config/              # Configuration loading
├── .github/workflows/       # CI/CD pipelines
├── docs/                    # Documentation
├── examples/                # Sample PRDs
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
- No database persistence
- No approval gates
- No session resume after crash
- macOS/Linux only (no Windows)
- Generated code may require minor fixes (verified to compile with `go build`)

## Troubleshooting

### "agentapi not found"

Install AgentAPI:
```bash
# Download from releases
curl -fsSL "https://github.com/coder/agentapi/releases/latest/download/agentapi-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')" -o agentapi
chmod +x agentapi
sudo mv agentapi /usr/local/bin/
```

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

- [AgentAPI](https://github.com/coder/agentapi) - HTTP wrapper for Claude Code
- [Claude Code](https://claude.ai/claude-code) - AI coding assistant
- [Cobra](https://github.com/spf13/cobra) - CLI framework
