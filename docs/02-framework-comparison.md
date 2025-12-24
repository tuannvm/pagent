# Framework Comparison: Claude Agent SDK vs Mastra vs VoltAgent

> **Implementation Note:** This document was created during the research phase. The final implementation chose [AgentAPI](https://github.com/coder/agentapi) over the Claude Agent SDK for simplicity. See [04-implementation-plan.md](./04-implementation-plan.md) for details.

## Overview

This document provides a detailed comparison of three TypeScript frameworks for building AI agent systems, based on research conducted in December 2025.

## Project Health Metrics

| Metric | Claude Agent SDK | Mastra | VoltAgent |
|--------|------------------|--------|-----------|
| **Stars** | 480 | 18,855 | 4,213 |
| **Forks** | N/A | 1,349 | 396 |
| **Open Issues** | N/A | 294 | 26 |
| **Created** | 2025 | Aug 2024 | Apr 2025 |
| **Last Commit** | Daily | Daily | Daily |
| **License** | Commercial ToS | Elastic 2.0 | MIT |
| **Backing** | Anthropic | Y Combinator W25 | Independent |
| **npm Package** | `@anthropic-ai/claude-agent-sdk` | `@mastra/core` | `@voltagent/core` |

## Architecture Comparison

### Claude Agent SDK

The official Anthropic SDK that exposes Claude Code as a library.

```typescript
import { query } from "@anthropic-ai/claude-agent-sdk";

for await (const message of query({
  prompt: "Find and fix the bug in auth.py",
  options: {
    allowedTools: ["Read", "Edit", "Bash"],
    hooks: {
      PostToolUse: [{ matcher: "Edit", hooks: [{ type: "command", command: "echo 'File edited'" }] }]
    }
  }
})) {
  console.log(message);
}
```

**Key characteristics:**
- Single `query()` function as entry point
- Async generator for streaming responses
- Built-in tool execution (no implementation needed)
- Hooks for lifecycle events
- Session management for persistence

### Mastra

A comprehensive TypeScript framework with its own agent abstraction.

```typescript
import { Agent, createWorkflow, createStep } from "@mastra/core";

const routingAgent = new Agent({
  id: "routing-agent",
  model: "openai/gpt-5.1",  // String-based model routing
  agents: { researchAgent, writingAgent },
  workflows: { cityWorkflow },
  tools: { weatherTool },
  memory: new Memory({ storage: new LibSQLStore({...}) }),
});

const workflow = createWorkflow({...})
  .then(step1)
  .then(step2)
  .commit();
```

**Key characteristics:**
- Agent Networks for multi-agent coordination
- Own model routing layer (40+ providers)
- Workflow engine with `.then()`, `.branch()`, `.parallel()`
- Storage abstraction (LibSQL, etc.)

### VoltAgent

TypeScript framework built on Vercel AI SDK with built-in observability.

```typescript
import { Agent } from "@voltagent/core";
import { openai } from "@ai-sdk/openai";

const supervisorAgent = new Agent({
  name: "supervisor-agent",
  instructions: "Coordinate specialists...",
  model: openai("gpt-4o-mini"),  // Vercel AI SDK model
  subAgents: [agentA, agentB, agentC],
});
```

**Key characteristics:**
- Supervisor + SubAgents pattern
- Delegates to Vercel AI SDK for LLM calls
- VoltOps for built-in observability
- Workflow chains with suspend/resume

## Feature Matrix

### Core Agent Features

| Feature | Claude Agent SDK | Mastra | VoltAgent |
|---------|------------------|--------|-----------|
| Agent Definition | `query()` function | `Agent` class | `Agent` class |
| Multi-Agent | `Task` tool | Agent Networks | Supervisor pattern |
| Model Support | Claude only | 40+ providers | Via Vercel AI SDK |
| Streaming | Native async generator | `stream()` method | `streamText()` |

### Built-in Tools

| Tool | Claude Agent SDK | Mastra | VoltAgent |
|------|------------------|--------|-----------|
| File Read | `Read` | Must implement | Must implement |
| File Write | `Write` | Must implement | Must implement |
| File Edit | `Edit` | Must implement | Must implement |
| Shell Commands | `Bash` | Must implement | Must implement |
| File Search | `Glob` | Must implement | Must implement |
| Content Search | `Grep` | Must implement | Must implement |
| Web Search | `WebSearch` | Must implement | Must implement |
| Web Fetch | `WebFetch` | Must implement | Must implement |

### Orchestration Features

| Feature | Claude Agent SDK | Mastra | VoltAgent |
|---------|------------------|--------|-----------|
| Subagent Spawning | `Task` tool | `agents` config | `subAgents` array |
| Workflow Engine | No (build your own) | `.then()/.branch()/.parallel()` | `createWorkflowChain()` |
| Deterministic Routing | No (LLM decides) | No (LLM decides) | No (LLM decides) |
| Parallel Execution | Multiple `query()` calls | `.parallel()` step | Parallel workflows |

### Human-in-the-Loop

| Feature | Claude Agent SDK | Mastra | VoltAgent |
|---------|------------------|--------|-----------|
| Suspend Workflow | Via hooks (block) | `suspend()` in step | `suspend()` in step |
| Resume Workflow | `resume` session option | `run.resume()` | `execution.resume()` |
| Tool Approval | Hooks + permission modes | `requireToolApproval` | Via workflow |
| Approval UI | Must build | Must build | Must build |

### Observability

| Feature | Claude Agent SDK | Mastra | VoltAgent |
|---------|------------------|--------|-----------|
| Tracing | Via hooks | OpenTelemetry | VoltOps (built-in) |
| Logging | Via hooks | PinoLogger | Built-in logger |
| Dashboard | Must build | Mastra Studio/Cloud | VoltOps Console |
| External Providers | Must implement | MLflow, Langfuse, etc. | Langfuse |

### Session Management

| Feature | Claude Agent SDK | Mastra | VoltAgent |
|---------|------------------|--------|-----------|
| Session Persistence | Native (`sessionId`) | Storage providers | Workflow state |
| Resume Session | `resume` option | `createRunAsync()` | `execution.resume()` |
| Fork Session | Supported | Via snapshots | Not documented |
| Cross-request State | Native | Via storage | Via memory adapters |

## Multi-Agent Patterns

### Claude Agent SDK: Task-based Delegation

```typescript
// Enable Task tool for subagent spawning
for await (const message of query({
  prompt: "Research quantum computing and write a report",
  options: {
    allowedTools: ["Read", "Write", "WebSearch", "Task"],
    // Claude decides when to spawn subagents
  }
})) {
  console.log(message);
}
```

Claude autonomously decides when to delegate to subagents based on task complexity.

### Mastra: Agent Networks

```typescript
const routingAgent = new Agent({
  id: "pm-agent",
  instructions: "Route to appropriate specialist...",
  agents: {
    designLead: designAgent,
    techLead: techAgent,
    qaLead: qaAgent,
  },
  memory: new Memory({...}),
});

// LLM decides routing based on agent descriptions
```

### VoltAgent: Supervisor Pattern

```typescript
const pmAgent = new Agent({
  name: "pm-supervisor",
  instructions: "Coordinate product development...",
  model: openai("gpt-4o"),
  subAgents: [designAgent, techAgent, qaAgent],
  supervisorConfig: {
    delegationStrategy: "auto",
  },
});
```

## Gap Analysis for PM Workflow

| Requirement | Claude Agent SDK | Mastra | VoltAgent |
|-------------|------------------|--------|-----------|
| PM controls specialists | Via `Task` tool | Agent Networks | Supervisor |
| PRD → TRD flow | Build workflow | Workflow engine | Workflow chains |
| Parallel execution | Multiple queries | `.parallel()` | Parallel workflows |
| Real-time observation | Hooks | Tracing | VoltOps |
| Step into session | `resume` + prompt | Not supported | Not supported |
| Approval gates | Hooks (block) | `suspend()` | `suspend()` |

## Decision Matrix

| If you need... | Use |
|----------------|-----|
| Claude Code's exact capabilities | Claude Agent SDK |
| Built-in file/shell tools | Claude Agent SDK |
| Multiple LLM providers | Mastra or VoltAgent |
| Complex workflow DAGs | Mastra |
| Built-in observability UI | VoltAgent |
| Official Anthropic support | Claude Agent SDK |
| MIT license | VoltAgent |
| Largest community | Mastra |

## Conclusion

For a PM workflow built specifically around Claude:

**Original Recommendation:** Claude Agent SDK was identified as the optimal choice for its native subagent support and built-in tools.

**Final Decision:** We chose [AgentAPI](https://github.com/coder/agentapi) instead for the v1 implementation because:

1. **Simpler architecture** - HTTP API calls vs SDK integration
2. **Process isolation** - Each agent runs as a separate process
3. **Faster development** - No TypeScript/Node.js dependency for a Go CLI
4. **Battle-tested** - AgentAPI is actively maintained by Coder with good community adoption

The trade-off is:
- No native hooks (must poll for status)
- No session persistence across runs
- Less fine-grained control over agent behavior

This is acceptable for v1 given the simplicity benefits. The Claude Agent SDK remains an option for v2 if more sophisticated orchestration is needed.
