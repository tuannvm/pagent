# Implementation Guide: pagent

## Overview

This document provides detailed implementation guidance for **pagent**, including code structure, key implementations, and technical decisions.

## Prerequisites

Before starting:

```bash
# Verify AgentAPI is installed
which agentapi  # Should return path

# Verify Claude Code is authenticated
claude --version  # Should work without auth errors

# Verify Go is installed
go version  # 1.21+ recommended
```

## Project Initialization

```bash
mkdir -p pagent
cd pagent
go mod init github.com/tuannvm/pagent

# Dependencies
go get gopkg.in/yaml.v3  # YAML parsing
```

## Directory Structure

```
pagent/
├── cmd/
│   └── pagent/
│       └── main.go
├── internal/
│   ├── config/
│   │   ├── config.go
│   │   └── defaults.go
│   ├── agent/
│   │   ├── agent.go
│   │   ├── client.go
│   │   └── status.go
│   ├── orchestrator/
│   │   ├── orchestrator.go
│   │   └── deps.go
│   └── output/
│       └── printer.go
├── configs/
│   └── default.yaml
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Core Types

### internal/agent/status.go

```go
package agent

type Status string

const (
    StatusPending   Status = "pending"
    StatusSpawning  Status = "spawning"
    StatusRunning   Status = "running"
    StatusStable    Status = "stable"
    StatusCompleted Status = "completed"
    StatusFailed    Status = "failed"
)

func (s Status) IsTerminal() bool {
    return s == StatusCompleted || s == StatusFailed
}

func (s Status) CanReceiveMessage() bool {
    return s == StatusStable
}
```

### internal/config/config.go

```go
package config

import (
    "fmt"
    "os"
    "path/filepath"

    "gopkg.in/yaml.v3"
)

type Config struct {
    OutputDir string                 `yaml:"output_dir"`
    Timeout   int                    `yaml:"timeout"` // seconds
    Agents    map[string]AgentConfig `yaml:"agents"`
}

type AgentConfig struct {
    Prompt    string   `yaml:"prompt"`
    Output    string   `yaml:"output"`
    DependsOn []string `yaml:"depends_on"`
}

const (
    ConfigDir  = ".pagent"
    ConfigFile = "config.yaml"
)

func Load() (*Config, error) {
    // Check for local config
    configPath := filepath.Join(ConfigDir, ConfigFile)
    if _, err := os.Stat(configPath); os.IsNotExist(err) {
        return DefaultConfig(), nil
    }

    data, err := os.ReadFile(configPath)
    if err != nil {
        return nil, fmt.Errorf("failed to read config: %w", err)
    }

    var cfg Config
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        return nil, fmt.Errorf("failed to parse config: %w", err)
    }

    // Apply environment overrides
    if dir := os.Getenv("PAGENT_OUTPUT_DIR"); dir != "" {
        cfg.OutputDir = dir
    }
    if timeout := os.Getenv("PAGENT_TIMEOUT"); timeout != "" {
        // Parse and set timeout
    }

    return &cfg, nil
}

func (c *Config) Save() error {
    if err := os.MkdirAll(ConfigDir, 0755); err != nil {
        return err
    }

    data, err := yaml.Marshal(c)
    if err != nil {
        return err
    }

    return os.WriteFile(filepath.Join(ConfigDir, ConfigFile), data, 0644)
}
```

### internal/config/defaults.go

```go
package config

func DefaultConfig() *Config {
    return &Config{
        OutputDir: "./outputs",
        Timeout:   300, // 5 minutes per agent
        Agents: map[string]AgentConfig{
            "design": {
                Prompt: `You are a Design Lead. Read the PRD at {prd_path}.
Create a design specification and write it to {output_path}.

Include:
- UI/UX requirements
- User flows (use mermaid diagrams)
- Component specifications
- Accessibility requirements

Be specific and actionable. Reference existing patterns when applicable.`,
                Output:    "design-spec.md",
                DependsOn: []string{},
            },
            "tech": {
                Prompt: `You are a Tech Lead. Read the PRD at {prd_path}.
Review any existing design docs in the outputs directory.
Create technical requirements and write them to {output_path}.

Include:
- Architecture decisions
- API specifications (endpoints, schemas)
- Data models
- Technical constraints
- Implementation approach

Be specific. Flag any PRD ambiguities.`,
                Output:    "technical-requirements.md",
                DependsOn: []string{"design"},
            },
            "qa": {
                Prompt: `You are a QA Lead. Read the PRD at {prd_path}.
Review existing specs in the outputs directory.
Create a test plan and write it to {output_path}.

Include:
- Test strategy
- Test cases (unit, integration, e2e)
- Acceptance criteria
- Edge cases to cover

Be thorough. Consider both happy paths and error scenarios.`,
                Output:    "test-plan.md",
                DependsOn: []string{"tech"},
            },
            "security": {
                Prompt: `You are a Security Reviewer. Read the PRD at {prd_path}.
Review the technical requirements in outputs directory.
Create a security assessment and write it to {output_path}.

Include:
- Threat model
- Security requirements
- Authentication/authorization needs
- Data protection requirements
- Risk mitigations

Follow OWASP guidelines where applicable.`,
                Output:    "security-assessment.md",
                DependsOn: []string{"tech"},
            },
            "infra": {
                Prompt: `You are an Infrastructure Lead. Read the PRD at {prd_path}.
Review the technical requirements in outputs directory.
Create an infrastructure plan and write it to {output_path}.

Include:
- Resource requirements
- Deployment strategy
- Scaling considerations
- Monitoring and alerting
- Cost estimates (if applicable)

Be practical and consider operational complexity.`,
                Output:    "infrastructure-plan.md",
                DependsOn: []string{"tech"},
            },
        },
    }
}

func DefaultAgentNames() []string {
    return []string{"design", "tech", "qa", "security", "infra"}
}
```

## AgentAPI Client

### internal/agent/client.go

```go
package agent

import (
    "bufio"
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strings"
    "time"
)

type Client struct {
    baseURL    string
    httpClient *http.Client
}

// Message represents a conversation message from AgentAPI
type Message struct {
    ID      int64  `json:"id"`
    Content string `json:"content"`
    Role    string `json:"role"` // "agent" or "user"
    Time    string `json:"time"`
}

// StatusResponse from GET /status
type StatusResponse struct {
    AgentType string `json:"agent_type"`
    Status    string `json:"status"` // "running" or "stable"
}

// MessagesResponse from GET /messages
type MessagesResponse struct {
    Messages []Message `json:"messages"`
}

// MessageRequest for POST /message
type MessageRequest struct {
    Content string `json:"content"`
    Type    string `json:"type"` // "user" or "raw"
}

// MessageResponse from POST /message
type MessageResponse struct {
    OK bool `json:"ok"`
}

// SSE Event types
type Event struct {
    EventType string          `json:"-"` // from "event:" line
    Data      json.RawMessage `json:"data"`
}

type MessageUpdateEvent struct {
    ID      int64  `json:"id"`
    Message string `json:"message"`
    Role    string `json:"role"`
    Time    string `json:"time"`
}

type StatusChangeEvent struct {
    AgentType string `json:"agent_type"`
    Status    string `json:"status"`
}

func NewClient(baseURL string) *Client {
    return &Client{
        baseURL: baseURL,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
}

// GetStatus returns the agent status ("running" or "stable")
func (c *Client) GetStatus() (string, error) {
    resp, err := c.httpClient.Get(c.baseURL + "/status")
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
    }

    var statusResp StatusResponse
    if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
        return "", fmt.Errorf("failed to parse status response: %w", err)
    }

    return statusResp.Status, nil
}

// SendMessage sends a message to the agent
func (c *Client) SendMessage(content string, msgType string) error {
    payload := map[string]string{
        "content": content,
        "type":    msgType,
    }

    data, err := json.Marshal(payload)
    if err != nil {
        return err
    }

    resp, err := c.httpClient.Post(
        c.baseURL+"/message",
        "application/json",
        bytes.NewReader(data),
    )
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("send message failed: %d - %s", resp.StatusCode, string(body))
    }

    return nil
}

// GetMessages returns conversation history
func (c *Client) GetMessages() ([]Message, error) {
    resp, err := c.httpClient.Get(c.baseURL + "/messages")
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var messagesResp MessagesResponse
    if err := json.NewDecoder(resp.Body).Decode(&messagesResp); err != nil {
        return nil, fmt.Errorf("failed to parse messages response: %w", err)
    }

    return messagesResp.Messages, nil
}

// StreamEvents opens SSE connection for real-time events
// Events are: "message_update" and "status_change"
func (c *Client) StreamEvents(ctx context.Context) (<-chan Event, error) {
    req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/events", nil)
    if err != nil {
        return nil, err
    }
    req.Header.Set("Accept", "text/event-stream")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }

    if resp.StatusCode != http.StatusOK {
        resp.Body.Close()
        return nil, fmt.Errorf("SSE connection failed: %d", resp.StatusCode)
    }

    events := make(chan Event, 100)

    go func() {
        defer resp.Body.Close()
        defer close(events)

        scanner := bufio.NewScanner(resp.Body)
        var currentEvent Event

        for scanner.Scan() {
            line := scanner.Text()

            if line == "" {
                // Empty line = end of event, dispatch if we have data
                if currentEvent.EventType != "" {
                    select {
                    case events <- currentEvent:
                    case <-ctx.Done():
                        return
                    }
                }
                currentEvent = Event{}
                continue
            }

            if strings.HasPrefix(line, "event: ") {
                currentEvent.EventType = strings.TrimPrefix(line, "event: ")
            } else if strings.HasPrefix(line, "data: ") {
                currentEvent.Data = json.RawMessage(strings.TrimPrefix(line, "data: "))
            }
        }
    }()

    return events, nil
}

// WaitForHealthy waits until the agent is responding
func (c *Client) WaitForHealthy(ctx context.Context, timeout time.Duration) error {
    deadline := time.Now().Add(timeout)
    ticker := time.NewTicker(500 * time.Millisecond)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            if time.Now().After(deadline) {
                return fmt.Errorf("health check timeout after %v", timeout)
            }
            if _, err := c.GetStatus(); err == nil {
                return nil
            }
        }
    }
}

// WaitForStable waits until agent status is "stable"
func (c *Client) WaitForStable(ctx context.Context, timeout time.Duration) error {
    deadline := time.Now().Add(timeout)
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            if time.Now().After(deadline) {
                return fmt.Errorf("timeout waiting for stable state")
            }
            status, err := c.GetStatus()
            if err == nil && status == "stable" {
                return nil
            }
        }
    }
}
```

## Agent Lifecycle

### internal/agent/agent.go

```go
package agent

import (
    "context"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strconv"
    "strings"
    "sync"
    "time"

    "github.com/tuannvm/pagent/internal/config"
)

type Agent struct {
    ID       string
    Config   config.AgentConfig
    Port     int
    Process  *exec.Cmd
    Client   *Client
    Status   Status
    WorkDir  string
    OutputDir string

    mu       sync.RWMutex
    cancelFn context.CancelFunc
}

func New(id string, cfg config.AgentConfig, port int, workDir, outputDir string) *Agent {
    return &Agent{
        ID:        id,
        Config:    cfg,
        Port:      port,
        Status:    StatusPending,
        WorkDir:   workDir,
        OutputDir: outputDir,
    }
}

func (a *Agent) Spawn(ctx context.Context) error {
    a.mu.Lock()
    defer a.mu.Unlock()

    a.Status = StatusSpawning

    // Create context with cancel for cleanup
    spawnCtx, cancel := context.WithCancel(ctx)
    a.cancelFn = cancel

    // Start agentapi server
    a.Process = exec.CommandContext(spawnCtx, "agentapi", "server",
        "--port", strconv.Itoa(a.Port),
        "--", "claude",
    )
    a.Process.Dir = a.WorkDir

    // Capture stderr for debugging
    a.Process.Stderr = os.Stderr

    if err := a.Process.Start(); err != nil {
        a.Status = StatusFailed
        return fmt.Errorf("failed to start agentapi: %w", err)
    }

    // Create client
    a.Client = NewClient(fmt.Sprintf("http://localhost:%d", a.Port))

    // Wait for health check
    healthCtx, healthCancel := context.WithTimeout(ctx, 30*time.Second)
    defer healthCancel()

    if err := a.Client.WaitForHealthy(healthCtx, 30*time.Second); err != nil {
        a.Stop()
        a.Status = StatusFailed
        return fmt.Errorf("health check failed: %w", err)
    }

    a.Status = StatusStable
    return nil
}

func (a *Agent) SendTask(prdPath string) error {
    a.mu.Lock()
    defer a.mu.Unlock()

    if a.Client == nil {
        return fmt.Errorf("agent not spawned")
    }

    // Build prompt with path substitutions
    outputPath := filepath.Join(a.OutputDir, a.Config.Output)
    prompt := a.Config.Prompt
    prompt = strings.ReplaceAll(prompt, "{prd_path}", prdPath)
    prompt = strings.ReplaceAll(prompt, "{output_path}", outputPath)

    a.Status = StatusRunning

    return a.Client.SendMessage(prompt, "user")
}

func (a *Agent) SendMessage(content string) error {
    a.mu.RLock()
    defer a.mu.RUnlock()

    if a.Client == nil {
        return fmt.Errorf("agent not spawned")
    }

    if a.Status != StatusStable {
        return fmt.Errorf("agent is not idle (status: %s)", a.Status)
    }

    return a.Client.SendMessage(content, "user")
}

func (a *Agent) WaitForCompletion(ctx context.Context, timeout time.Duration) error {
    deadline := time.Now().Add(timeout)
    ticker := time.NewTicker(2 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            if time.Now().After(deadline) {
                a.mu.Lock()
                a.Status = StatusFailed
                a.mu.Unlock()
                return fmt.Errorf("timeout after %v", timeout)
            }

            status, err := a.Client.GetStatus()
            if err != nil {
                continue
            }

            a.mu.Lock()
            if status == "stable" {
                // Check if output file exists
                outputPath := filepath.Join(a.OutputDir, a.Config.Output)
                if _, err := os.Stat(outputPath); err == nil {
                    a.Status = StatusCompleted
                    a.mu.Unlock()
                    return nil
                }
                a.Status = StatusStable
            } else {
                a.Status = StatusRunning
            }
            a.mu.Unlock()
        }
    }
}

func (a *Agent) Stop() error {
    a.mu.Lock()
    defer a.mu.Unlock()

    if a.cancelFn != nil {
        a.cancelFn()
    }

    if a.Process != nil && a.Process.Process != nil {
        // Try graceful shutdown first
        a.Process.Process.Signal(os.Interrupt)

        // Wait briefly, then force kill
        done := make(chan error, 1)
        go func() { done <- a.Process.Wait() }()

        select {
        case <-done:
        case <-time.After(5 * time.Second):
            a.Process.Process.Kill()
        }
    }

    return nil
}

func (a *Agent) GetStatus() Status {
    a.mu.RLock()
    defer a.mu.RUnlock()
    return a.Status
}

func (a *Agent) GetMessages() ([]Message, error) {
    a.mu.RLock()
    defer a.mu.RUnlock()

    if a.Client == nil {
        return nil, fmt.Errorf("agent not spawned")
    }

    return a.Client.GetMessages()
}
```

## Orchestrator

### internal/orchestrator/deps.go

```go
package orchestrator

import (
    "fmt"

    "github.com/tuannvm/pagent/internal/config"
)

// TopologicalSort returns agents in dependency order
func TopologicalSort(agents map[string]config.AgentConfig, selected []string) ([]string, error) {
    // Build selected set
    selectedSet := make(map[string]bool)
    for _, name := range selected {
        selectedSet[name] = true
    }

    // Build dependency graph (only for selected agents)
    deps := make(map[string][]string)
    for name := range selectedSet {
        cfg, ok := agents[name]
        if !ok {
            return nil, fmt.Errorf("unknown agent: %s", name)
        }
        // Only include dependencies that are also selected
        var filteredDeps []string
        for _, dep := range cfg.DependsOn {
            if selectedSet[dep] {
                filteredDeps = append(filteredDeps, dep)
            }
        }
        deps[name] = filteredDeps
    }

    // Kahn's algorithm for topological sort
    inDegree := make(map[string]int)
    for name := range selectedSet {
        inDegree[name] = 0
    }
    for _, dependencies := range deps {
        for _, dep := range dependencies {
            inDegree[dep]++
        }
    }

    // Find all nodes with no incoming edges
    var queue []string
    for name, degree := range inDegree {
        if degree == 0 {
            queue = append(queue, name)
        }
    }

    var result []string
    for len(queue) > 0 {
        // Pop from queue
        node := queue[0]
        queue = queue[1:]
        result = append(result, node)

        // Reduce in-degree for dependents
        for name, dependencies := range deps {
            for _, dep := range dependencies {
                if dep == node {
                    inDegree[name]--
                    if inDegree[name] == 0 {
                        queue = append(queue, name)
                    }
                }
            }
        }
    }

    if len(result) != len(selected) {
        return nil, fmt.Errorf("circular dependency detected")
    }

    // Reverse to get correct order (dependencies first)
    for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
        result[i], result[j] = result[j], result[i]
    }

    return result, nil
}
```

### internal/orchestrator/orchestrator.go

```go
package orchestrator

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "sync"
    "syscall"
    "time"

    "github.com/tuannvm/pagent/internal/agent"
    "github.com/tuannvm/pagent/internal/config"
)

type Orchestrator struct {
    config   *config.Config
    agents   map[string]*agent.Agent
    basePort int
    mu       sync.RWMutex
}

type Result struct {
    Agent   string
    Success bool
    Output  string
    Error   error
}

func New(cfg *config.Config) *Orchestrator {
    return &Orchestrator{
        config:   cfg,
        agents:   make(map[string]*agent.Agent),
        basePort: 3284,
    }
}

func (o *Orchestrator) Run(ctx context.Context, prdPath string, agentNames []string, sequential bool) ([]Result, error) {
    // Validate PRD exists
    if _, err := os.Stat(prdPath); os.IsNotExist(err) {
        return nil, fmt.Errorf("PRD file not found: %s", prdPath)
    }

    // Create output directory
    if err := os.MkdirAll(o.config.OutputDir, 0755); err != nil {
        return nil, fmt.Errorf("failed to create output dir: %w", err)
    }

    // Get working directory
    workDir, err := os.Getwd()
    if err != nil {
        return nil, err
    }

    // Setup signal handling for cleanup
    ctx, cancel := context.WithCancel(ctx)
    defer cancel()

    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
    go func() {
        <-sigCh
        fmt.Println("\nReceived interrupt, cleaning up...")
        cancel()
        o.StopAll()
    }()

    if sequential {
        return o.runSequential(ctx, prdPath, agentNames, workDir)
    }
    return o.runParallel(ctx, prdPath, agentNames, workDir)
}

func (o *Orchestrator) runParallel(ctx context.Context, prdPath string, agentNames []string, workDir string) ([]Result, error) {
    var wg sync.WaitGroup
    resultCh := make(chan Result, len(agentNames))

    for i, name := range agentNames {
        cfg, ok := o.config.Agents[name]
        if !ok {
            resultCh <- Result{Agent: name, Success: false, Error: fmt.Errorf("unknown agent")}
            continue
        }

        port := o.basePort + i
        ag := agent.New(name, cfg, port, workDir, o.config.OutputDir)

        o.mu.Lock()
        o.agents[name] = ag
        o.mu.Unlock()

        wg.Add(1)
        go func(name string, ag *agent.Agent) {
            defer wg.Done()
            result := o.runSingleAgent(ctx, ag, prdPath)
            resultCh <- result
        }(name, ag)

        fmt.Printf("✓ %s: spawning on port %d...\n", name, port)
    }

    // Wait for all agents
    go func() {
        wg.Wait()
        close(resultCh)
    }()

    // Collect results
    var results []Result
    for result := range resultCh {
        results = append(results, result)
        if result.Success {
            fmt.Printf("✓ %s: completed → %s\n", result.Agent, result.Output)
        } else {
            fmt.Printf("✗ %s: failed (%v)\n", result.Agent, result.Error)
        }
    }

    return results, nil
}

func (o *Orchestrator) runSequential(ctx context.Context, prdPath string, agentNames []string, workDir string) ([]Result, error) {
    // Sort by dependencies
    sorted, err := TopologicalSort(o.config.Agents, agentNames)
    if err != nil {
        return nil, err
    }

    var results []Result

    for i, name := range sorted {
        select {
        case <-ctx.Done():
            return results, ctx.Err()
        default:
        }

        cfg := o.config.Agents[name]
        port := o.basePort + i
        ag := agent.New(name, cfg, port, workDir, o.config.OutputDir)

        o.mu.Lock()
        o.agents[name] = ag
        o.mu.Unlock()

        fmt.Printf("✓ %s: spawning on port %d...\n", name, port)

        result := o.runSingleAgent(ctx, ag, prdPath)
        results = append(results, result)

        if result.Success {
            fmt.Printf("✓ %s: completed → %s\n", result.Agent, result.Output)
        } else {
            fmt.Printf("✗ %s: failed (%v)\n", result.Agent, result.Error)
            // Continue with other agents even on failure
        }
    }

    return results, nil
}

func (o *Orchestrator) runSingleAgent(ctx context.Context, ag *agent.Agent, prdPath string) Result {
    result := Result{Agent: ag.ID}

    // Spawn agent
    if err := ag.Spawn(ctx); err != nil {
        result.Error = fmt.Errorf("spawn failed: %w", err)
        return result
    }

    // Send task
    if err := ag.SendTask(prdPath); err != nil {
        ag.Stop()
        result.Error = fmt.Errorf("send task failed: %w", err)
        return result
    }

    // Wait for completion
    timeout := time.Duration(o.config.Timeout) * time.Second
    if err := ag.WaitForCompletion(ctx, timeout); err != nil {
        ag.Stop()
        result.Error = err
        return result
    }

    result.Success = true
    result.Output = o.config.OutputDir + "/" + ag.Config.Output
    return result
}

func (o *Orchestrator) GetAgent(name string) *agent.Agent {
    o.mu.RLock()
    defer o.mu.RUnlock()
    return o.agents[name]
}

func (o *Orchestrator) GetAllAgents() map[string]*agent.Agent {
    o.mu.RLock()
    defer o.mu.RUnlock()

    result := make(map[string]*agent.Agent)
    for k, v := range o.agents {
        result[k] = v
    }
    return result
}

func (o *Orchestrator) StopAll() {
    o.mu.Lock()
    defer o.mu.Unlock()

    for name, ag := range o.agents {
        if err := ag.Stop(); err != nil {
            fmt.Printf("Warning: failed to stop %s: %v\n", name, err)
        }
    }
}

func (o *Orchestrator) Stop(name string) error {
    o.mu.Lock()
    defer o.mu.Unlock()

    ag, ok := o.agents[name]
    if !ok {
        return fmt.Errorf("agent not found: %s", name)
    }

    return ag.Stop()
}
```

## CLI Commands

### cmd/pagent/main.go

```go
package main

import (
    "context"
    "fmt"
    "os"
    "strings"
    "time"

    "github.com/tuannvm/pagent/internal/config"
    "github.com/tuannvm/pagent/internal/orchestrator"
)

var version = "0.1.0"

func main() {
    if len(os.Args) < 2 {
        printUsage()
        os.Exit(1)
    }

    cmd := os.Args[1]

    switch cmd {
    case "run":
        cmdRun(os.Args[2:])
    case "status":
        cmdStatus(os.Args[2:])
    case "logs":
        cmdLogs(os.Args[2:])
    case "message":
        cmdMessage(os.Args[2:])
    case "stop":
        cmdStop(os.Args[2:])
    case "init":
        cmdInit(os.Args[2:])
    case "agents":
        cmdAgents(os.Args[2:])
    case "--version", "-v":
        fmt.Printf("pagent %s\n", version)
    case "--help", "-h", "help":
        printUsage()
    default:
        fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
        printUsage()
        os.Exit(1)
    }
}

func printUsage() {
    fmt.Println(`pagent - PM Agent Workflow CLI

Usage:
  pagent <command> [options]

Commands:
  run <prd>              Run specialists on a PRD
  status                 Show agent status
  logs <agent>           View agent conversation
  message <agent> <msg>  Send message to idle agent
  stop [agent|--all]     Stop agent(s)
  init                   Create config file
  agents list            List available agents
  agents show <agent>    Show agent prompt

Options:
  --help, -h             Show help
  --version, -v          Show version

Examples:
  pagent run ./prd.md
  pagent run ./prd.md --agents design,tech
  pagent run ./prd.md --sequential
  pagent status
  pagent message design "Focus on mobile UX"
  pagent stop --all`)
}

func cmdRun(args []string) {
    if len(args) < 1 {
        fmt.Fprintln(os.Stderr, "Error: PRD path required")
        fmt.Fprintln(os.Stderr, "Usage: pagent run <prd> [--agents a,b] [--sequential] [--output dir]")
        os.Exit(1)
    }

    prdPath := args[0]
    var agentNames []string
    sequential := false
    outputDir := ""

    // Parse flags
    for i := 1; i < len(args); i++ {
        switch args[i] {
        case "--agents", "-a":
            if i+1 < len(args) {
                agentNames = strings.Split(args[i+1], ",")
                i++
            }
        case "--sequential", "-s":
            sequential = true
        case "--output", "-o":
            if i+1 < len(args) {
                outputDir = args[i+1]
                i++
            }
        }
    }

    // Load config
    cfg, err := config.Load()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
        os.Exit(1)
    }

    // Override output dir if specified
    if outputDir != "" {
        cfg.OutputDir = outputDir
    }

    // Default to all agents
    if len(agentNames) == 0 {
        agentNames = config.DefaultAgentNames()
    }

    // Validate agent names
    for _, name := range agentNames {
        if _, ok := cfg.Agents[name]; !ok {
            fmt.Fprintf(os.Stderr, "Error: unknown agent '%s'\n", name)
            fmt.Fprintf(os.Stderr, "Available agents: %v\n", config.DefaultAgentNames())
            os.Exit(1)
        }
    }

    fmt.Printf("Running %d agents on %s\n", len(agentNames), prdPath)
    if sequential {
        fmt.Println("Mode: sequential (respecting dependencies)")
    } else {
        fmt.Println("Mode: parallel")
    }
    fmt.Println()

    // Run orchestrator
    ctx := context.Background()
    orch := orchestrator.New(cfg)

    startTime := time.Now()
    results, err := orch.Run(ctx, prdPath, agentNames, sequential)
    elapsed := time.Since(startTime)

    fmt.Println()
    fmt.Println("─────────────────────────────────────")
    fmt.Printf("Completed in %v\n", elapsed.Round(time.Second))

    // Summary
    succeeded := 0
    for _, r := range results {
        if r.Success {
            succeeded++
        }
    }

    if succeeded == len(agentNames) {
        fmt.Printf("✓ All %d agents succeeded\n", succeeded)
    } else {
        fmt.Printf("⚠ %d/%d agents succeeded\n", succeeded, len(agentNames))
    }

    fmt.Printf("\nOutputs saved to: %s/\n", cfg.OutputDir)

    if err != nil || succeeded < len(agentNames) {
        os.Exit(1)
    }
}

func cmdStatus(args []string) {
    // For v1, this would need a running orchestrator
    // or a way to discover running agentapi processes
    fmt.Println("Agent status:")
    fmt.Println("  (No agents currently running)")
    fmt.Println()
    fmt.Println("Tip: Use 'pagent run <prd>' to start agents")
}

func cmdLogs(args []string) {
    if len(args) < 1 {
        fmt.Fprintln(os.Stderr, "Error: agent name required")
        fmt.Fprintln(os.Stderr, "Usage: pagent logs <agent>")
        os.Exit(1)
    }

    agentName := args[0]
    fmt.Printf("Logs for agent '%s':\n", agentName)
    fmt.Println("  (Agent not currently running)")
}

func cmdMessage(args []string) {
    if len(args) < 2 {
        fmt.Fprintln(os.Stderr, "Error: agent name and message required")
        fmt.Fprintln(os.Stderr, "Usage: pagent message <agent> \"message\"")
        os.Exit(1)
    }

    agentName := args[0]
    message := strings.Join(args[1:], " ")

    fmt.Printf("Sending message to '%s': %s\n", agentName, message)
    fmt.Println("  (Agent not currently running)")
}

func cmdStop(args []string) {
    if len(args) == 0 {
        fmt.Fprintln(os.Stderr, "Error: specify agent name or --all")
        fmt.Fprintln(os.Stderr, "Usage: pagent stop <agent> | --all")
        os.Exit(1)
    }

    if args[0] == "--all" {
        fmt.Println("Stopping all agents...")
        fmt.Println("  (No agents currently running)")
    } else {
        fmt.Printf("Stopping agent '%s'...\n", args[0])
        fmt.Println("  (Agent not currently running)")
    }
}

func cmdInit(args []string) {
    cfg := config.DefaultConfig()

    if err := cfg.Save(); err != nil {
        fmt.Fprintf(os.Stderr, "Error creating config: %v\n", err)
        os.Exit(1)
    }

    fmt.Printf("Created %s/%s\n", config.ConfigDir, config.ConfigFile)
    fmt.Println("Edit this file to customize agent prompts and settings.")
}

func cmdAgents(args []string) {
    if len(args) == 0 {
        fmt.Fprintln(os.Stderr, "Usage: pagent agents <list|show> [agent]")
        os.Exit(1)
    }

    cfg, _ := config.Load()

    switch args[0] {
    case "list":
        fmt.Println("Available agents:")
        for name, ac := range cfg.Agents {
            deps := "none"
            if len(ac.DependsOn) > 0 {
                deps = strings.Join(ac.DependsOn, ", ")
            }
            fmt.Printf("  %-12s → %s (depends: %s)\n", name, ac.Output, deps)
        }

    case "show":
        if len(args) < 2 {
            fmt.Fprintln(os.Stderr, "Usage: pagent agents show <agent>")
            os.Exit(1)
        }
        name := args[1]
        ac, ok := cfg.Agents[name]
        if !ok {
            fmt.Fprintf(os.Stderr, "Unknown agent: %s\n", name)
            os.Exit(1)
        }
        fmt.Printf("Agent: %s\n", name)
        fmt.Printf("Output: %s\n", ac.Output)
        fmt.Printf("Depends on: %v\n", ac.DependsOn)
        fmt.Println("\nPrompt:")
        fmt.Println("─────────────────────────────────────")
        fmt.Println(ac.Prompt)

    default:
        fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n", args[0])
        os.Exit(1)
    }
}
```

## Build & Install

### Makefile

```makefile
.PHONY: build install clean test

VERSION := 0.1.0
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o bin/pagent ./cmd/pagent

install: build
	cp bin/pagent /usr/local/bin/

clean:
	rm -rf bin/

test:
	go test -v ./...

lint:
	golangci-lint run

run-example:
	./bin/pagent run examples/sample-prd.md --agents design
```

## Testing

### internal/agent/client_test.go

```go
package agent

import (
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"
)

func TestClientGetStatus(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path == "/status" {
            w.Header().Set("Content-Type", "application/json")
            w.Write([]byte(`{"agent_type":"claude","status":"stable"}`))
        }
    }))
    defer server.Close()

    client := NewClient(server.URL)
    status, err := client.GetStatus()
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if status != "stable" {
        t.Errorf("expected 'stable', got '%s'", status)
    }
}

func TestClientSendMessage(t *testing.T) {
    var receivedContent string
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path == "/message" && r.Method == "POST" {
            var msg map[string]string
            json.NewDecoder(r.Body).Decode(&msg)
            receivedContent = msg["content"]
            w.WriteHeader(http.StatusOK)
        }
    }))
    defer server.Close()

    client := NewClient(server.URL)
    err := client.SendMessage("test message", "user")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if receivedContent != "test message" {
        t.Errorf("expected 'test message', got '%s'", receivedContent)
    }
}

func TestClientWaitForStable(t *testing.T) {
    callCount := 0
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        callCount++
        w.Header().Set("Content-Type", "application/json")
        if callCount < 3 {
            w.Write([]byte(`{"agent_type":"claude","status":"running"}`))
        } else {
            w.Write([]byte(`{"agent_type":"claude","status":"stable"}`))
        }
    }))
    defer server.Close()

    client := NewClient(server.URL)
    ctx := context.Background()

    err := client.WaitForStable(ctx, 10*time.Second)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if callCount < 3 {
        t.Errorf("expected at least 3 calls, got %d", callCount)
    }
}
```

## Error Handling Patterns

### Common Error Scenarios

```go
// Check AgentAPI installation
func checkAgentAPI() error {
    _, err := exec.LookPath("agentapi")
    if err != nil {
        return fmt.Errorf("agentapi not found in PATH. Install from: https://github.com/coder/agentapi")
    }
    return nil
}

// Check Claude Code authentication
func checkClaudeAuth() error {
    cmd := exec.Command("claude", "--version")
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("claude code not accessible. Run 'claude' to authenticate")
    }
    return nil
}

// Preflight checks before run
func preflight() error {
    if err := checkAgentAPI(); err != nil {
        return err
    }
    if err := checkClaudeAuth(); err != nil {
        return err
    }
    return nil
}
```

## Next Steps After MVP

1. **Persistent status** - Track running agents across CLI invocations
2. **Better SSE handling** - Real-time output streaming to terminal
3. **Approval gates** - Claude Code hooks integration
4. **Session resume** - Pick up where interrupted
5. **Cost tracking** - If AgentAPI exposes token counts
