# Tutorial

## Prerequisites

1. **Claude Code** installed and authenticated (`claude --version`)
2. **pagent** CLI:
   ```bash
   # Homebrew (recommended)
   brew install tuannvm/mcp/pagent

   # Or download binary
   curl -sSL https://github.com/tuannvm/pagent/releases/latest/download/pagent_$(uname -s | tr '[:upper:]' '[:lower:]')_$(uname -m | sed 's/x86_64/amd64/').tar.gz | tar xz
   sudo mv pagent /usr/local/bin/

   # Or from source (requires Go 1.21+)
   git clone https://github.com/tuannvm/pagent && cd pagent && make install
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

# 2. Launch the TUI
pagent ui

# 3. Select your PRD, choose a persona, and hit Run
```

## Using the TUI

The TUI is the primary way to interact with pagent. Launch it with:

```bash
pagent ui                    # Start fresh
pagent ui ./prd.md           # Pre-fill with a specific PRD
pagent ui --accessible       # Screen reader support
```

### Main Screen

When you launch the TUI, you'll see:

```
 ██████╗  █████╗  ██████╗ ███████╗███╗   ██╗████████╗
 ██╔══██╗██╔══██╗██╔════╝ ██╔════╝████╗  ██║╚══██╔══╝
 ██████╔╝███████║██║  ███╗█████╗  ██╔██╗ ██║   ██║
 ██╔═══╝ ██╔══██║██║   ██║██╔══╝  ██║╚██╗██║   ██║
 ██║     ██║  ██║╚██████╔╝███████╗██║ ╚████║   ██║
 ╚═╝     ╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚═╝  ╚═══╝   ╚═╝

 From idea to implementation, orchestrated.

 Input
 Select PRD or input file
 > 📁 examples/
   📁 specs/
   README.md
   examples/sample-prd.md

 Persona
   Minimal - MVP focus
 > Balanced - Standard
   Production - Enterprise

 Output
 > ./outputs

 Action
 Shift+Tab go back
 > ▶ Run
   ⚙ Advanced...
   × Cancel

 ↑ up · ↓ down · / filter · enter select
```

**Fields:**

| Field | Description |
|-------|-------------|
| Input | PRD or spec file. Auto-discovers `*.md`, `*.yaml`, `prd*`, `requirements*` |
| Persona | `minimal` (MVP), `balanced` (default), `production` (enterprise) |
| Output | Where generated files go |

**Navigation:** `Tab`/`Shift+Tab` to move, `Enter` to select, `Space` to toggle

### File Discovery

The Input dropdown auto-discovers:
- Recent markdown and YAML files
- Common folders: `inputs/`, `examples/`, `specs/`
- A "🔍 Browse..." option to open the file picker

File picker controls:
- `Enter` - Open/select
- `.` - Toggle hidden files
- `Esc` - Cancel

### Advanced Settings

Select **⚙ Advanced...** to configure:

| Setting | Options |
|---------|---------|
| Agents | Multi-select which agents to run |
| Mode | `parallel` (default) or `sequential` |
| Resume | `normal`, `resume` (skip up-to-date), `force` (overwrite all) |
| Architecture | `config`, `stateless`, `database` |
| Timeout | Seconds before timeout |
| Config | Path to custom config file |
| Verbosity | `normal`, `verbose`, `quiet` |

Press `Esc` to return to the main screen.

### Personas

| Persona | Use Case |
|---------|----------|
| `minimal` | MVP, prototype - ship fast |
| `balanced` | Standard projects - maintainable |
| `production` | Enterprise - comprehensive testing, security |

## Agents

| Phase | Agent | Output |
|-------|-------|--------|
| Spec | architect | `architecture.md` - API design, data models |
| Spec | qa | `test-plan.md` - Test cases, acceptance criteria |
| Spec | security | `security-assessment.md` - Threat model |
| Impl | implementer | `code/*` - Complete codebase |
| Impl | verifier | `code/*_test.go` - Tests, validation report |

## Execution Modes

**Parallel (default):** Agents run concurrently within dependency levels
```
Level 0: architect
Level 1: qa, security (parallel)
Level 2: implementer
Level 3: verifier
```

**Sequential:** One agent at a time, strict order

**Resume:** Skip agents with up-to-date outputs (SHA-256 hashing)

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

The TUI reads these defaults automatically.

## CLI Reference

For scripting or CI/CD, use the CLI directly:

### `pagent run`

```bash
pagent run ./prd.md                    # All agents, parallel
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
```

## Workflows

### Quick Specs Review
1. `pagent ui ./prd.md`
2. Advanced → Select only `architect`, `qa`, `security`
3. Run
4. Review outputs, iterate on PRD

### Full Pipeline
1. `pagent ui ./prd.md`
2. Select `production` persona
3. Run with all agents
4. `cd outputs/code && go build ./...`

### Iterative Development
1. `pagent ui` → Run architect only
2. Review `architecture.md`
3. `pagent ui` → Run remaining agents with Resume mode

## Troubleshooting

| Issue | Fix |
|-------|-----|
| `agentapi not found` | Install from [coder/agentapi](https://github.com/coder/agentapi/releases) |
| Timeout | Increase in Advanced settings or `--timeout 600` |
| Port in use | `pagent stop --all` or `lsof -i :3284-3290 \| awk 'NR>1 {print $2}' \| xargs kill` |
| Incomplete output | `pagent message <agent> "Please complete..."` |
| TUI not rendering | Try `--accessible` flag or check terminal compatibility |
