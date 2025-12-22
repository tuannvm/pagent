# Implementation Plan: pagent

## Overview

This document outlines the implementation plan for **pagent**, a minimal CLI orchestrator for PM agent workflows built on AgentAPI.

**Target**: ~500-800 lines of Go code, 6-8 days to MVP.

## Architecture Recap

```
┌─────────────────────────────────────────────────┐
│              pagent (Go CLI)                     │
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
```

## Phase 1: Project Setup & Core Types (Day 1)

### Goals
- Initialize Go module
- Define core data structures
- Set up project layout

### Deliverables

```
pagent/
├── cmd/
│   └── pagent/
│       └── main.go           # CLI entry point
├── internal/
│   ├── config/
│   │   └── config.go         # YAML config loader
│   ├── agent/
│   │   ├── agent.go          # Agent lifecycle management
│   │   └── client.go         # AgentAPI HTTP client
│   ├── orchestrator/
│   │   └── orchestrator.go   # Multi-agent coordination
│   └── cli/
│       └── commands.go       # CLI command implementations
├── configs/
│   └── default.yaml          # Default agent prompts
├── go.mod
├── go.sum
└── README.md
```

### Tasks
- [ ] `go mod init github.com/tuannvm/pagent`
- [ ] Define `AgentConfig` struct
- [ ] Define `Config` struct for YAML loading
- [ ] Create `AgentStatus` enum (pending, spawning, running, stable, completed, failed)

## Phase 2: AgentAPI Client (Day 2)

### Goals
- Implement HTTP client for AgentAPI
- Handle all required endpoints

### AgentAPI Endpoints

| Endpoint | Method | Response Format | Purpose |
|----------|--------|-----------------|---------|
| `/status` | GET | `{"agent_type":"claude","status":"running\|stable"}` | Check agent state |
| `/messages` | GET | `{"messages":[{id,content,role,time},...]}` | Get conversation history |
| `/message` | POST | `{"ok":true}` | Send message (requires `stable` state for `user` type) |
| `/events` | GET | SSE: `message_update`, `status_change` events | Real-time updates |
| `/upload` | POST | `{"ok":true,"filePath":"..."}` | Upload files (multipart/form-data) |

### Tasks
- [ ] Implement `Client` struct with base URL and HTTP client
- [ ] `GetStatus() (string, error)` - returns "running" or "stable"
- [ ] `SendMessage(content string, msgType string) error`
- [ ] `GetMessages() ([]Message, error)`
- [ ] `StreamEvents(ctx context.Context) (<-chan Event, error)` - SSE client
- [ ] Add health check with retry logic

### Key Code

```go
type Client struct {
    baseURL    string
    httpClient *http.Client
}

func (c *Client) WaitForStable(ctx context.Context, timeout time.Duration) error {
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        status, err := c.GetStatus()
        if err == nil && status == "stable" {
            return nil
        }
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(500 * time.Millisecond):
        }
    }
    return fmt.Errorf("timeout waiting for stable state")
}
```

## Phase 3: Agent Lifecycle (Day 3)

### Goals
- Spawn AgentAPI processes
- Manage lifecycle (start, monitor, stop)
- Handle cleanup on interrupt

### Tasks
- [ ] Implement `Agent` struct with process handle
- [ ] `Spawn(port int, workDir string) error` - start AgentAPI process
- [ ] `Stop() error` - graceful shutdown
- [ ] `SendTask(prompt string) error` - send initial task
- [ ] Port allocation (3284-3290)
- [ ] Health check before sending tasks
- [ ] SIGINT/SIGTERM handling for cleanup

### Key Code

```go
type Agent struct {
    ID       string
    Config   AgentConfig
    Port     int
    Process  *exec.Cmd
    Client   *Client
    Status   AgentStatus
    WorkDir  string
}

func (a *Agent) Spawn(ctx context.Context) error {
    a.Process = exec.CommandContext(ctx, "agentapi", "server",
        "--port", strconv.Itoa(a.Port),
        "--", "claude")
    a.Process.Dir = a.WorkDir

    if err := a.Process.Start(); err != nil {
        return fmt.Errorf("failed to start agentapi: %w", err)
    }

    a.Client = NewClient(fmt.Sprintf("http://localhost:%d", a.Port))

    // Wait for health check
    if err := a.Client.WaitForHealthy(ctx, 30*time.Second); err != nil {
        a.Process.Process.Kill()
        return err
    }

    return nil
}
```

## Phase 4: Orchestrator (Day 4)

### Goals
- Coordinate multiple agents
- Handle parallel vs sequential execution
- Dependency management

### Tasks
- [ ] Implement `Orchestrator` struct
- [ ] `RunAll(prdPath string, agents []string) error`
- [ ] `RunSequential(prdPath string, agents []string) error`
- [ ] Dependency resolution (topological sort)
- [ ] Progress reporting
- [ ] Partial failure handling

### Agent Dependencies

```
design ────────────────────────────┐
    │                              │
    ▼                              │
  tech ───────┬─────────┐          │
    │         │         │          │
    ▼         ▼         ▼          │
   qa     security    infra        │
    │         │         │          │
    └─────────┴─────────┴──────────┘
```

### Key Code

```go
type Orchestrator struct {
    config    *Config
    agents    map[string]*Agent
    portAlloc *PortAllocator
}

func (o *Orchestrator) RunParallel(ctx context.Context, prdPath string, agentNames []string) error {
    var wg sync.WaitGroup
    errCh := make(chan error, len(agentNames))

    for _, name := range agentNames {
        wg.Add(1)
        go func(name string) {
            defer wg.Done()
            if err := o.runAgent(ctx, name, prdPath); err != nil {
                errCh <- fmt.Errorf("%s: %w", name, err)
            }
        }(name)
    }

    wg.Wait()
    close(errCh)

    // Collect errors
    var errs []error
    for err := range errCh {
        errs = append(errs, err)
    }

    if len(errs) > 0 {
        return fmt.Errorf("partial failure: %d/%d agents failed", len(errs), len(agentNames))
    }
    return nil
}
```

## Phase 5: CLI Commands (Day 5)

### Goals
- Implement all CLI commands
- Add flags and options
- Help text and usage

### Commands

| Command | Description |
|---------|-------------|
| `pagent run <prd>` | Run all specialists on PRD |
| `pagent run <prd> --agents a,b` | Run specific agents |
| `pagent run <prd> --sequential` | Run with dependency ordering |
| `pagent status` | Show running agents |
| `pagent logs <agent>` | View agent conversation |
| `pagent message <agent> "msg"` | Send message to idle agent |
| `pagent stop [agent\|--all]` | Stop agent(s) |
| `pagent init` | Create config file |

### Tasks
- [ ] Choose CLI framework (cobra or std flag)
- [ ] Implement `run` command with all flags
- [ ] Implement `status` command
- [ ] Implement `logs` command
- [ ] Implement `message` command
- [ ] Implement `stop` command
- [ ] Implement `init` command
- [ ] Add `--help` for all commands
- [ ] Add `--verbose` and `--quiet` flags

## Phase 6: Configuration (Day 6)

### Goals
- YAML config file support
- Default prompts for all agents
- Environment variable overrides

### Config File Location
- `.pagent/config.yaml` in current directory
- Falls back to defaults if not present

### Tasks
- [ ] YAML parsing with `gopkg.in/yaml.v3`
- [ ] Default config generation
- [ ] Environment variable overrides (`PAGENT_OUTPUT_DIR`, `PAGENT_TIMEOUT`)
- [ ] Config validation

### Default Prompts

```yaml
agents:
  design:
    prompt: |
      You are a Design Lead. Read the PRD at {prd_path}.
      Create a design specification at {output_path}.

      Include:
      - UI/UX requirements
      - User flows (mermaid diagrams)
      - Component specifications
      - Accessibility requirements
    output: design-spec.md
    depends_on: []

  tech:
    prompt: |
      You are a Tech Lead. Read the PRD at {prd_path}.
      Create technical requirements at {output_path}.

      Include:
      - Architecture decisions
      - API specifications
      - Data models
      - Technical constraints
    output: technical-requirements.md
    depends_on: [design]
```

## Phase 7: Testing & Polish (Days 7-8)

### Goals
- Unit tests for core components
- Integration tests
- Error handling improvements
- Documentation

### Tasks
- [ ] Unit tests for AgentAPI client
- [ ] Unit tests for config loader
- [ ] Integration test with mock AgentAPI
- [ ] Error message improvements
- [ ] README with usage examples
- [ ] `--version` flag

### Test Coverage Targets

| Package | Coverage |
|---------|----------|
| `internal/config` | 90% |
| `internal/agent` | 80% |
| `internal/orchestrator` | 70% |
| `cmd/pagent` | 60% |

## Milestones

### M1: Single Agent (End of Day 3)
- [ ] Can spawn one AgentAPI instance
- [ ] Can send task and wait for completion
- [ ] Clean shutdown on Ctrl+C

### M2: Multi-Agent (End of Day 5)
- [ ] All 5 agents defined with prompts
- [ ] Parallel execution works
- [ ] Progress output during execution

### M3: Full CLI (End of Day 6)
- [ ] All commands implemented
- [ ] Config file support
- [ ] `--help` works everywhere

### M4: Production Ready (End of Day 8)
- [ ] Tests passing
- [ ] README complete
- [ ] Release binary built

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| AgentAPI not in PATH | Blocks startup | Check on init, clear error message |
| Port conflicts | Agent spawn fails | Try alternate ports, report which port |
| Agent hangs | Never completes | Configurable timeout, `stop` command |
| Claude Code auth fails | All agents fail | Check auth before spawning |

## Out of Scope (v1)

- Web dashboard
- Database persistence
- Approval gates
- Session resume
- Cost tracking
- Windows support

## Success Criteria

1. `pagent run prd.md` produces 5 markdown files in `outputs/`
2. All agents complete within configured timeout
3. Clean shutdown on Ctrl+C with no orphan processes
4. `pagent status` accurately shows agent states
5. `pagent message` successfully injects guidance
