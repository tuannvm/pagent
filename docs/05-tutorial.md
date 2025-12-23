# User Tutorial: PM Agent Workflow

This tutorial walks you through using `pm-agents` to transform a PRD into comprehensive deliverables.

## Prerequisites

Before starting, ensure you have:

1. **AgentAPI** installed:
   ```bash
   # Download and install
   curl -fsSL "https://github.com/coder/agentapi/releases/latest/download/agentapi-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')" -o agentapi
   chmod +x agentapi
   sudo mv agentapi /usr/local/bin/

   # Verify installation
   agentapi --version
   ```

2. **Claude Code** installed and authenticated:
   ```bash
   # Verify Claude Code is working
   claude --version
   ```

3. **pm-agents** binary:
   ```bash
   # Build from source
   git clone https://github.com/tuannvm/pm-agent-workflow
   cd pm-agent-workflow
   make build

   # Or install to PATH
   make install
   ```

## Quick Start

### Step 1: Create Your PRD

Create a markdown file with your product requirements:

```bash
cat > my-prd.md << 'EOF'
# Product Requirements: My App

## Overview
Build a simple todo application...

## Features
- User authentication
- Task CRUD operations
- ...
EOF
```

### Step 2: Run All Agents

```bash
pm-agents run ./my-prd.md
```

This spawns 5 specialist agents in dependency order:

**Specification Phase:**
- **architect** - Creates comprehensive architecture document (API, data models, design)
- **qa** - Creates test plan and acceptance criteria
- **security** - Creates threat model and security requirements

**Implementation Phase:**
- **implementer** - Creates complete codebase (API, database, migrations)
- **verifier** - Validates implementation and writes tests

### Step 3: Check Progress

While agents are running, monitor their status:

```bash
# Check which agents are running
pm-agents status

# Output:
# AGENT     PORT  STATUS
# design    3284  running
# tech      3285  stable
# qa        3286  running
```

### Step 4: View Agent Logs

See what an agent is doing:

```bash
pm-agents logs design
```

### Step 5: Send Guidance (Optional)

If an agent needs redirection:

```bash
pm-agents message design "Focus more on mobile UX"
```

### Step 6: Collect Outputs

When complete, find deliverables in `./outputs/`:

```bash
ls outputs/
# design-spec.md
# technical-requirements.md
# test-plan.md
# security-assessment.md
# infrastructure-plan.md
# code/                      # Generated codebase (from dev agents)
```

If you ran developer agents, check the generated code:

```bash
ls outputs/code/
# cmd/server/main.go
# internal/handler/*.go
# internal/service/*.go
# internal/repository/*.go
# migrations/*.sql
# go.mod, Dockerfile, Makefile

# Verify it compiles
cd outputs/code && go mod tidy && go build ./...
```

## Command Reference

### `pm-agents run`

Run specialist agents on a PRD.

```bash
# Run all 5 agents in dependency order (recommended)
pm-agents run ./prd.md --sequential

# Run spec agents only (documentation)
pm-agents run ./prd.md --agents architect,qa,security --sequential

# Run implementation agents only (requires specs to exist)
pm-agents run ./prd.md --agents implementer,verifier --sequential

# Run specific agent
pm-agents run ./prd.md --agents architect

# Custom output directory
pm-agents run ./prd.md --output ./docs/specs/

# With verbose output
pm-agents run ./prd.md -v

# Custom timeout (seconds per agent)
pm-agents run ./prd.md --timeout 600
```

### `pm-agents status`

Check status of running agents.

```bash
pm-agents status
```

### `pm-agents logs`

View agent conversation history.

```bash
pm-agents logs <agent-name>

# Examples:
pm-agents logs design
pm-agents logs tech
```

### `pm-agents message`

Send guidance to an idle agent.

```bash
pm-agents message <agent-name> "<message>"

# Examples:
pm-agents message design "Add dark mode support"
pm-agents message tech "Use GraphQL instead of REST"
```

### `pm-agents stop`

Stop running agents.

```bash
# Stop specific agent
pm-agents stop design

# Stop all agents
pm-agents stop --all
```

### `pm-agents init`

Initialize configuration in current directory.

```bash
pm-agents init
# Creates .pm-agents/config.yaml
```

### `pm-agents agents`

List and show agent definitions.

```bash
# List all agents
pm-agents agents list

# Show agent prompt template
pm-agents agents show design
```

## Configuration

### Default Configuration

Run `pm-agents init` to create `.pm-agents/config.yaml`:

```yaml
output_dir: ./outputs
timeout: 300  # seconds per agent

# Implementation style: minimal, balanced, production
persona: balanced

# Architecture preferences
preferences:
  stateless: false          # true = event-driven, false = database-backed
  api_style: rest           # rest, graphql, grpc
  language: go              # go, python, typescript, java, rust
  testing_depth: unit       # none, unit, integration, e2e
  documentation_level: standard
  dependency_style: minimal       # prefer Go stdlib over third-party
  containerized: true
  include_ci: true
  include_iac: true

# Technology stack
stack:
  cloud: aws
  compute: kubernetes
  database: postgres
  cache: redis
  iac: terraform
  gitops: argocd
  ci: github-actions
  monitoring: prometheus

# Agent customization (prompts loaded from embedded templates)
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

Choose an implementation style based on your needs:

| Persona | Use Case | Characteristics |
|---------|----------|-----------------|
| `minimal` | MVP, prototype | Ship fast, simple code, skip observability |
| `balanced` | Growing product | Essential quality, maintainable code |
| `production` | Enterprise | Comprehensive testing, security, observability |

```bash
# Override persona via CLI
pm-agents run prd.md --persona minimal
```

### Customizing Technology Stack

Tailor the stack to your infrastructure:

```yaml
# Example: GCP + MongoDB setup
stack:
  cloud: gcp
  compute: gke
  database: mongodb
  cache: memcached
  message_queue: pubsub
  iac: terraform
  gitops: flux
  ci: gitlab-ci
  monitoring: datadog
```

### Prompt Variables

Available variables in prompts:

| Variable | Description |
|----------|-------------|
| `{prd_path}` | Absolute path to the PRD file |
| `{output_path}` | Absolute path to the output file |

### Environment Variables

Override settings via environment:

```bash
# Override output directory
PM_AGENTS_OUTPUT_DIR=./docs pm-agents run prd.md

# Override timeout
PM_AGENTS_TIMEOUT=600 pm-agents run prd.md
```

## Execution Modes

### Parallel Mode (Default)

Agents run concurrently within dependency levels:

```bash
pm-agents run ./prd.md
```

**How it works:**
```
Level 0: architect (no dependencies)
Level 1: qa, security (both run in parallel)
Level 2: implementer
Level 3: verifier
```

- Each level must complete before the next starts
- Faster than sequential while respecting dependencies
- Best for most workflows

### Sequential Mode

Agents run one at a time in dependency order:

```bash
pm-agents run ./prd.md --sequential
```

- Strictest ordering: architect → qa → security → implementer → verifier
- Slowest but most predictable
- Useful for debugging or constrained resources

### Resume Mode

Skip agents whose outputs are already up-to-date:

```bash
pm-agents run ./prd.md --resume
```

**Change detection uses content hashing (SHA-256):**
- Input files changed?
- Configuration (persona, stack, preferences) changed?
- Dependency outputs changed?

```bash
# Force regeneration of all outputs
pm-agents run ./prd.md --force
```

## Troubleshooting

### "agentapi not found"

Install AgentAPI:
```bash
curl -fsSL "https://github.com/coder/agentapi/releases/latest/download/agentapi-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m)" -o agentapi
chmod +x agentapi
sudo mv agentapi /usr/local/bin/
```

### "timeout waiting for agent"

- Increase timeout: `--timeout 600`
- Check Claude Code authentication: `claude --version`
- Check agent logs: `pm-agents logs <agent>`

### "port already in use"

Kill existing processes:
```bash
pm-agents stop --all

# Or manually:
lsof -i :3284-3290 | awk 'NR>1 {print $2}' | xargs kill
```

### Agent produces incomplete output

Send additional guidance:
```bash
pm-agents message design "Please complete the component specifications section"
```

### Want to restart a specific agent

```bash
pm-agents stop design
pm-agents run ./prd.md --agents design
```

## Example Workflows

### Workflow 1: Documentation Only

Generate specification documents from a PRD:

```bash
# 1. Initialize project
mkdir my-project && cd my-project
pm-agents init

# 2. Create PRD
cat > prd.md << 'EOF'
# Product Requirements: Task Manager API
...
EOF

# 3. Run spec agents only
pm-agents run ./prd.md --agents architect,qa,security --sequential -v

# 4. Check outputs
ls outputs/
# architecture.md, test-plan.md, security-assessment.md
```

### Workflow 2: Full PRD-to-Code Pipeline

Generate specs AND working code from a PRD:

```bash
# 1. Run all 5 agents in sequential mode (recommended)
pm-agents run ./prd.md --sequential -v

# 2. Monitor progress (in another terminal)
pm-agents status

# 3. Check generated outputs
ls outputs/
# Spec docs:
#   architecture.md, test-plan.md, security-assessment.md
# Verification:
#   verification-report.md
# Generated code:
#   code/

# 4. Verify generated code compiles
cd outputs/code
go mod tidy
go build ./...
go vet ./...

# 5. Review the generated structure
find . -type f -name "*.go" | head -20
```

### Workflow 3: Iterative Development

Run spec agents first, review, then generate code:

```bash
# Step 1: Generate architecture
pm-agents run ./prd.md --agents architect --sequential -v

# Step 2: Review and provide feedback
pm-agents message architect "Use Chi router instead of standard library"

# Step 3: Run remaining agents after architecture is finalized
pm-agents run ./prd.md --agents qa,security,implementer,verifier --sequential -v

# Step 4: Verify
cd outputs/code && go build ./...
```

## Tips

1. **Use parallel mode** (default) for most workflows - it respects dependencies while being faster
2. **Use `--resume`** during iteration to skip up-to-date agents
3. **Choose the right persona**: `minimal` for prototypes, `balanced` for most projects, `production` for enterprise
4. **Customize the stack** to match your infrastructure (cloud, database, CI/CD)
5. **Use verbose mode** (`-v`) to see what's happening
6. **Check logs** if an agent seems stuck (`pm-agents logs <agent>`)
7. **Increase timeout** for complex tasks (`--timeout 600`)

## Next Steps

- Customize agent prompts for your domain
- Create PRD templates for common project types
- Integrate with your CI/CD pipeline
- Build on top of the generated artifacts
