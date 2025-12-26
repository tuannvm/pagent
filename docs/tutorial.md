# Tutorial

## Prerequisites

1. **Claude Code** installed and authenticated (`claude --version`)
2. **pagent** CLI:
   ```bash
   # From source
   git clone https://github.com/tuannvm/pagent && cd pagent && make install

   # Or download release binary
   ```

## Quick Start

```bash
# 1. Create a PRD
cat > prd.md << 'EOF'
# Product: Task Manager API
## Features
- User authentication (JWT)
- Task CRUD with categories
- Due date reminders
EOF

# 2. Run all agents
pagent run ./prd.md --sequential

# 3. Check outputs
ls outputs/
# architecture.md  test-plan.md  security-assessment.md
# verification-report.md  code/
```

## Agents

| Phase | Agent | Output |
|-------|-------|--------|
| Spec | architect | `architecture.md` - API design, data models |
| Spec | qa | `test-plan.md` - Test cases, acceptance criteria |
| Spec | security | `security-assessment.md` - Threat model |
| Impl | implementer | `code/*` - Complete codebase |
| Impl | verifier | `code/*_test.go` - Tests, validation report |

## Commands

### `pagent run`

```bash
pagent run ./prd.md                    # All agents, parallel by dependency level
pagent run ./prd.md --sequential       # Strict sequential order
pagent run ./prd.md --agents architect # Single agent
pagent run ./prd.md --resume           # Skip up-to-date outputs
pagent run ./prd.md --force            # Regenerate all
pagent run ./prd.md -o ./docs/ -v      # Custom output, verbose
```

### Other Commands

```bash
pagent status              # Check running agents
pagent logs <agent>        # View agent conversation
pagent message <agent> "..." # Send guidance to idle agent
pagent stop --all          # Stop all agents
pagent init                # Create .pagent/config.yaml
pagent agents list         # List available agents
pagent ui                  # Interactive TUI dashboard
```

## Configuration

Run `pagent init` to create `.pagent/config.yaml`:

```yaml
output_dir: ./outputs
timeout: 300

persona: balanced  # minimal | balanced | production

preferences:
  api_style: rest        # rest | graphql | grpc
  language: go           # go | python | typescript
  testing_depth: unit    # none | unit | integration | e2e
  containerized: true
  include_ci: true

stack:
  cloud: aws
  compute: kubernetes
  database: postgres
  cache: redis
```

### Personas

| Persona | Use Case |
|---------|----------|
| `minimal` | MVP, prototype - ship fast |
| `balanced` | Standard projects - maintainable |
| `production` | Enterprise - comprehensive testing, security |

## Execution Modes

**Parallel (default)**: Agents run concurrently within dependency levels
```
Level 0: architect
Level 1: qa, security (parallel)
Level 2: implementer
Level 3: verifier
```

**Sequential** (`--sequential`): One agent at a time, strict order

**Resume** (`--resume`): Skip agents with up-to-date outputs (uses SHA-256 hashing)

## Workflows

### Specs Only
```bash
pagent run ./prd.md --agents architect,qa,security --sequential
```

### Full Pipeline
```bash
pagent run ./prd.md --sequential -v
cd outputs/code && go build ./...  # Verify generated code
```

### Iterative
```bash
pagent run ./prd.md --agents architect
# Review architecture.md, then:
pagent run ./prd.md --agents qa,security,implementer,verifier --sequential
```

## Troubleshooting

| Issue | Fix |
|-------|-----|
| `agentapi not found` | Install from [coder/agentapi](https://github.com/coder/agentapi/releases) |
| Timeout | Increase with `--timeout 600` |
| Port in use | `pagent stop --all` or `lsof -i :3284-3290 \| awk 'NR>1 {print $2}' \| xargs kill` |
| Incomplete output | `pagent message <agent> "Please complete..."` |
