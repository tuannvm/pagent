# Implementation Plan: PM Agent Workflow System

## Overview

This document outlines the implementation plan for building a PM Agent Workflow System using the Claude Agent SDK. The system enables a Product Manager to orchestrate multiple specialist agents that produce deliverables from a PRD.

## Technology Stack

### Core
- **Runtime**: Node.js 20+ (LTS)
- **Language**: TypeScript 5.x
- **Agent SDK**: `@anthropic-ai/claude-agent-sdk`
- **Package Manager**: pnpm

### Backend
- **Framework**: Hono (lightweight, edge-compatible)
- **WebSocket**: ws (real-time streaming)
- **Database**: SQLite via better-sqlite3 (session persistence)
- **Queue**: BullMQ with Redis (job management)

### Frontend
- **Framework**: Next.js 14 (App Router)
- **UI Components**: shadcn/ui
- **State Management**: Zustand
- **Real-time**: Socket.io client

### DevOps
- **Containerization**: Docker
- **Local Development**: Docker Compose

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Dashboard (Next.js)                       │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────────┐   │
│  │ PRD View │ │ Agents   │ │ Approvals│ │ Outputs          │   │
│  │          │ │ Monitor  │ │ Queue    │ │ Viewer           │   │
│  └──────────┘ └──────────┘ └──────────┘ └──────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ WebSocket + REST
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Orchestration Server (Hono)                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │ REST API     │  │ WebSocket    │  │ Session Manager      │  │
│  │ /api/...     │  │ Server       │  │                      │  │
│  └──────────────┘  └──────────────┘  └──────────────────────┘  │
│                              │                                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │ Agent        │  │ Approval     │  │ Output               │  │
│  │ Manager      │  │ Manager      │  │ Manager              │  │
│  └──────────────┘  └──────────────┘  └──────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ Claude Agent SDK
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Agent Runtime Layer                         │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌───────────┐ │
│  │ Design Lead │ │ Tech Lead   │ │ QA Lead     │ │ Security  │ │
│  │ Agent       │ │ Agent       │ │ Agent       │ │ Agent     │ │
│  └─────────────┘ └─────────────┘ └─────────────┘ └───────────┘ │
│                                                                  │
│  Each agent runs as a separate query() with its own session     │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Persistence Layer                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │ SQLite DB    │  │ File System  │  │ Redis                │  │
│  │ (sessions)   │  │ (outputs)    │  │ (jobs/pubsub)        │  │
│  └──────────────┘  └──────────────┘  └──────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

## Project Structure

```
pm-agent-workflow/
├── apps/
│   ├── server/                 # Orchestration server
│   │   ├── src/
│   │   │   ├── index.ts        # Entry point
│   │   │   ├── api/            # REST endpoints
│   │   │   │   ├── projects.ts
│   │   │   │   ├── agents.ts
│   │   │   │   └── approvals.ts
│   │   │   ├── ws/             # WebSocket handlers
│   │   │   │   └── handlers.ts
│   │   │   ├── agents/         # Agent definitions
│   │   │   │   ├── base.ts
│   │   │   │   ├── design-lead.ts
│   │   │   │   ├── tech-lead.ts
│   │   │   │   ├── qa-lead.ts
│   │   │   │   ├── security-reviewer.ts
│   │   │   │   └── infra-lead.ts
│   │   │   ├── services/       # Business logic
│   │   │   │   ├── agent-manager.ts
│   │   │   │   ├── session-manager.ts
│   │   │   │   ├── approval-manager.ts
│   │   │   │   └── output-manager.ts
│   │   │   ├── hooks/          # Claude SDK hooks
│   │   │   │   ├── observation.ts
│   │   │   │   └── approval.ts
│   │   │   └── db/             # Database
│   │   │       ├── schema.ts
│   │   │       └── client.ts
│   │   └── package.json
│   │
│   └── dashboard/              # Next.js frontend
│       ├── src/
│       │   ├── app/
│       │   │   ├── page.tsx
│       │   │   ├── projects/
│       │   │   ├── agents/
│       │   │   └── approvals/
│       │   ├── components/
│       │   │   ├── agent-card.tsx
│       │   │   ├── agent-stream.tsx
│       │   │   ├── approval-dialog.tsx
│       │   │   └── output-viewer.tsx
│       │   ├── hooks/
│       │   │   └── use-agent-stream.ts
│       │   └── lib/
│       │       ├── api.ts
│       │       └── ws.ts
│       └── package.json
│
├── packages/
│   └── shared/                 # Shared types and utilities
│       ├── src/
│       │   ├── types.ts
│       │   └── constants.ts
│       └── package.json
│
├── .claude/                    # Claude Code configuration
│   ├── agents/                 # Agent definitions (markdown)
│   │   ├── design-lead.md
│   │   ├── tech-lead.md
│   │   ├── qa-lead.md
│   │   ├── security-reviewer.md
│   │   └── infra-lead.md
│   └── commands/               # Slash commands
│       └── pm-workflow.md
│
├── docs/                       # Documentation
├── docker-compose.yml
├── package.json
├── pnpm-workspace.yaml
└── tsconfig.json
```

## Implementation Phases

### Phase 1: Foundation (Week 1-2)

#### 1.1 Project Setup
- [ ] Initialize monorepo with pnpm workspaces
- [ ] Configure TypeScript, ESLint, Prettier
- [ ] Set up Docker Compose for local development
- [ ] Create shared types package

#### 1.2 Basic Agent Runtime
- [ ] Implement base agent wrapper around Claude Agent SDK
- [ ] Create agent configuration types
- [ ] Implement single agent execution flow
- [ ] Add basic logging

**Key Code: Base Agent Wrapper**

```typescript
// apps/server/src/agents/base.ts
import { query, ClaudeAgentOptions } from "@anthropic-ai/claude-agent-sdk";
import { EventEmitter } from "events";

export interface AgentConfig {
  id: string;
  name: string;
  role: string;
  systemPrompt: string;
  allowedTools: string[];
  model?: "opus" | "sonnet" | "haiku";
}

export interface AgentMessage {
  type: "text" | "tool_use" | "tool_result" | "system";
  content: any;
  timestamp: Date;
}

export class AgentRunner extends EventEmitter {
  private sessionId: string | null = null;
  private isRunning = false;
  private isPaused = false;

  constructor(
    private config: AgentConfig,
    private projectPath: string
  ) {
    super();
  }

  async run(prompt: string): Promise<void> {
    this.isRunning = true;
    this.emit("started", { agentId: this.config.id });

    const options: ClaudeAgentOptions = {
      allowedTools: this.config.allowedTools,
      systemPrompt: this.config.systemPrompt,
      workingDirectory: this.projectPath,
      hooks: {
        PreToolUse: [
          {
            matcher: ".*",
            hooks: [
              {
                type: "callback",
                callback: (toolUse) => this.handlePreToolUse(toolUse),
              },
            ],
          },
        ],
        PostToolUse: [
          {
            matcher: ".*",
            hooks: [
              {
                type: "callback",
                callback: (toolUse, result) =>
                  this.handlePostToolUse(toolUse, result),
              },
            ],
          },
        ],
      },
    };

    if (this.sessionId) {
      options.resume = this.sessionId;
    }

    try {
      for await (const message of query({ prompt, options })) {
        if (message.type === "system" && message.subtype === "init") {
          this.sessionId = message.session_id;
          this.emit("session", { sessionId: this.sessionId });
        }

        this.emit("message", {
          agentId: this.config.id,
          message: this.normalizeMessage(message),
        });

        // Check for pause request
        if (this.isPaused) {
          this.emit("paused", { agentId: this.config.id });
          await this.waitForResume();
        }
      }

      this.emit("completed", { agentId: this.config.id });
    } catch (error) {
      this.emit("error", { agentId: this.config.id, error });
    } finally {
      this.isRunning = false;
    }
  }

  pause(): void {
    this.isPaused = true;
  }

  resume(newPrompt?: string): void {
    this.isPaused = false;
    if (newPrompt) {
      // Inject new prompt into session
      this.run(newPrompt);
    }
  }

  private handlePreToolUse(toolUse: any): boolean {
    this.emit("toolUse", {
      agentId: this.config.id,
      phase: "pre",
      tool: toolUse.name,
      input: toolUse.input,
    });

    // Return true to allow, false to block
    // This is where approval gates would be implemented
    return true;
  }

  private handlePostToolUse(toolUse: any, result: any): void {
    this.emit("toolUse", {
      agentId: this.config.id,
      phase: "post",
      tool: toolUse.name,
      input: toolUse.input,
      output: result,
    });
  }

  private normalizeMessage(message: any): AgentMessage {
    return {
      type: message.type,
      content: message,
      timestamp: new Date(),
    };
  }

  private waitForResume(): Promise<void> {
    return new Promise((resolve) => {
      const check = () => {
        if (!this.isPaused) {
          resolve();
        } else {
          setTimeout(check, 100);
        }
      };
      check();
    });
  }
}
```

#### 1.3 Session Persistence
- [ ] Set up SQLite database schema
- [ ] Implement session CRUD operations
- [ ] Add session resume capability

**Database Schema**

```sql
-- apps/server/src/db/schema.sql
CREATE TABLE projects (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  prd_path TEXT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE agent_sessions (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL,
  agent_type TEXT NOT NULL,
  claude_session_id TEXT,
  status TEXT DEFAULT 'pending', -- pending, running, paused, completed, error
  started_at DATETIME,
  completed_at DATETIME,
  FOREIGN KEY (project_id) REFERENCES projects(id)
);

CREATE TABLE agent_messages (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  session_id TEXT NOT NULL,
  message_type TEXT NOT NULL,
  content TEXT NOT NULL,
  timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (session_id) REFERENCES agent_sessions(id)
);

CREATE TABLE approvals (
  id TEXT PRIMARY KEY,
  session_id TEXT NOT NULL,
  tool_name TEXT NOT NULL,
  tool_input TEXT NOT NULL,
  status TEXT DEFAULT 'pending', -- pending, approved, denied
  decided_at DATETIME,
  decided_by TEXT,
  FOREIGN KEY (session_id) REFERENCES agent_sessions(id)
);

CREATE TABLE outputs (
  id TEXT PRIMARY KEY,
  session_id TEXT NOT NULL,
  file_path TEXT NOT NULL,
  content TEXT NOT NULL,
  version INTEGER DEFAULT 1,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (session_id) REFERENCES agent_sessions(id)
);
```

### Phase 2: Agent Definitions (Week 2-3)

#### 2.1 Specialist Agent Configurations
- [ ] Design Lead agent definition
- [ ] Tech Lead agent definition
- [ ] QA Lead agent definition
- [ ] Security Reviewer agent definition
- [ ] Infra Lead agent definition

**Example: Tech Lead Agent**

```typescript
// apps/server/src/agents/tech-lead.ts
import { AgentConfig } from "./base";

export const techLeadConfig: AgentConfig = {
  id: "tech-lead",
  name: "Tech Lead",
  role: "Technical Lead responsible for creating TRD",
  allowedTools: ["Read", "Write", "Edit", "Glob", "Grep", "Bash"],
  model: "sonnet",
  systemPrompt: `You are a Senior Technical Lead responsible for creating Technical Requirements Documents (TRD) from Product Requirements Documents (PRD).

## Your Responsibilities
1. Analyze the PRD thoroughly
2. Review existing codebase architecture
3. Create a comprehensive TRD

## Output Format
Create a file at \`outputs/technical-requirements.md\` with:

# Technical Requirements Document

## 1. Overview
Brief summary of the technical approach

## 2. Architecture
- System architecture decisions
- Component diagram (mermaid)
- Data flow

## 3. API Specifications
- Endpoints
- Request/Response schemas
- Authentication requirements

## 4. Data Models
- Database schema
- Entity relationships

## 5. Technical Constraints
- Performance requirements
- Scalability considerations
- Security requirements

## 6. Implementation Approach
- Phased rollout plan
- Dependencies
- Risk mitigation

## 7. Effort Estimation
- Component breakdown
- Complexity assessment

## Guidelines
- Be specific and actionable
- Reference existing code patterns when possible
- Consider backward compatibility
- Flag any PRD ambiguities for PM review`,
};
```

**Markdown Agent Definition (for Claude Code)**

```markdown
<!-- .claude/agents/tech-lead.md -->
---
name: tech-lead
description: Technical Lead for TRD creation
tools:
  - Read
  - Write
  - Edit
  - Glob
  - Grep
  - Bash
model: sonnet
---

You are a Senior Technical Lead responsible for creating Technical Requirements Documents (TRD) from Product Requirements Documents (PRD).

## Your Responsibilities
1. Analyze the PRD thoroughly
2. Review existing codebase architecture
3. Create a comprehensive TRD

## Output Location
Always write your TRD to: `outputs/technical-requirements.md`

## TRD Structure
Follow this structure for your output:

1. **Overview** - Brief summary of technical approach
2. **Architecture** - System design, components, data flow
3. **API Specifications** - Endpoints, schemas, auth
4. **Data Models** - Database schema, relationships
5. **Technical Constraints** - Performance, security, scalability
6. **Implementation Approach** - Phases, dependencies, risks
7. **Effort Estimation** - Breakdown by component

## Guidelines
- Be specific and actionable
- Reference existing code patterns
- Consider backward compatibility
- Flag PRD ambiguities for PM review
```

#### 2.2 Agent Manager Service
- [ ] Implement multi-agent coordination
- [ ] Add dependency management between agents
- [ ] Implement parallel execution

```typescript
// apps/server/src/services/agent-manager.ts
import { EventEmitter } from "events";
import { AgentRunner, AgentConfig } from "../agents/base";
import { designLeadConfig } from "../agents/design-lead";
import { techLeadConfig } from "../agents/tech-lead";
import { qaLeadConfig } from "../agents/qa-lead";
import { securityReviewerConfig } from "../agents/security-reviewer";
import { infraLeadConfig } from "../agents/infra-lead";

interface WorkflowConfig {
  projectId: string;
  projectPath: string;
  prdContent: string;
  agents: string[];
  parallel: boolean;
}

interface AgentDependency {
  agent: string;
  dependsOn: string[];
}

const AGENT_CONFIGS: Record<string, AgentConfig> = {
  "design-lead": designLeadConfig,
  "tech-lead": techLeadConfig,
  "qa-lead": qaLeadConfig,
  "security-reviewer": securityReviewerConfig,
  "infra-lead": infraLeadConfig,
};

const AGENT_DEPENDENCIES: AgentDependency[] = [
  { agent: "design-lead", dependsOn: [] },
  { agent: "tech-lead", dependsOn: ["design-lead"] },
  { agent: "qa-lead", dependsOn: ["design-lead", "tech-lead"] },
  { agent: "security-reviewer", dependsOn: ["tech-lead"] },
  { agent: "infra-lead", dependsOn: ["tech-lead"] },
];

export class AgentManager extends EventEmitter {
  private runners: Map<string, AgentRunner> = new Map();
  private completedAgents: Set<string> = new Set();

  async runWorkflow(config: WorkflowConfig): Promise<void> {
    const { projectId, projectPath, prdContent, agents, parallel } = config;

    // Filter to requested agents
    const selectedDeps = AGENT_DEPENDENCIES.filter((d) =>
      agents.includes(d.agent)
    );

    if (parallel) {
      await this.runParallel(selectedDeps, projectPath, prdContent);
    } else {
      await this.runSequential(selectedDeps, projectPath, prdContent);
    }
  }

  private async runParallel(
    deps: AgentDependency[],
    projectPath: string,
    prdContent: string
  ): Promise<void> {
    const pending = new Set(deps.map((d) => d.agent));

    while (pending.size > 0) {
      // Find agents whose dependencies are satisfied
      const ready = deps.filter(
        (d) =>
          pending.has(d.agent) &&
          d.dependsOn.every((dep) => this.completedAgents.has(dep))
      );

      if (ready.length === 0 && pending.size > 0) {
        throw new Error("Circular dependency detected");
      }

      // Run ready agents in parallel
      await Promise.all(
        ready.map(async (dep) => {
          await this.runAgent(dep.agent, projectPath, prdContent);
          this.completedAgents.add(dep.agent);
          pending.delete(dep.agent);
        })
      );
    }
  }

  private async runSequential(
    deps: AgentDependency[],
    projectPath: string,
    prdContent: string
  ): Promise<void> {
    // Topological sort
    const sorted = this.topologicalSort(deps);

    for (const agentId of sorted) {
      await this.runAgent(agentId, projectPath, prdContent);
      this.completedAgents.add(agentId);
    }
  }

  private async runAgent(
    agentId: string,
    projectPath: string,
    prdContent: string
  ): Promise<void> {
    const config = AGENT_CONFIGS[agentId];
    if (!config) {
      throw new Error(`Unknown agent: ${agentId}`);
    }

    const runner = new AgentRunner(config, projectPath);
    this.runners.set(agentId, runner);

    // Forward events
    runner.on("started", (e) => this.emit("agentStarted", e));
    runner.on("message", (e) => this.emit("agentMessage", e));
    runner.on("toolUse", (e) => this.emit("agentToolUse", e));
    runner.on("completed", (e) => this.emit("agentCompleted", e));
    runner.on("error", (e) => this.emit("agentError", e));

    const prompt = this.buildPrompt(agentId, prdContent);
    await runner.run(prompt);
  }

  private buildPrompt(agentId: string, prdContent: string): string {
    return `# Task Assignment

You have been assigned to work on this project as the ${AGENT_CONFIGS[agentId].name}.

## Product Requirements Document

${prdContent}

## Your Assignment

Please analyze the PRD and create your deliverable according to your role's responsibilities.
Save your output to the designated file in the outputs/ directory.`;
  }

  pauseAgent(agentId: string): void {
    const runner = this.runners.get(agentId);
    if (runner) {
      runner.pause();
    }
  }

  resumeAgent(agentId: string, newPrompt?: string): void {
    const runner = this.runners.get(agentId);
    if (runner) {
      runner.resume(newPrompt);
    }
  }

  private topologicalSort(deps: AgentDependency[]): string[] {
    const result: string[] = [];
    const visited = new Set<string>();
    const temp = new Set<string>();

    const visit = (agent: string) => {
      if (temp.has(agent)) throw new Error("Circular dependency");
      if (visited.has(agent)) return;

      temp.add(agent);
      const dep = deps.find((d) => d.agent === agent);
      if (dep) {
        for (const d of dep.dependsOn) {
          visit(d);
        }
      }
      temp.delete(agent);
      visited.add(agent);
      result.push(agent);
    };

    for (const dep of deps) {
      visit(dep.agent);
    }

    return result;
  }
}
```

### Phase 3: Approval System (Week 3-4)

#### 3.1 Approval Manager
- [ ] Implement approval queue
- [ ] Add approval rules engine
- [ ] Create approval hooks for Claude SDK

```typescript
// apps/server/src/services/approval-manager.ts
import { EventEmitter } from "events";
import { db } from "../db/client";

export interface ApprovalRequest {
  id: string;
  sessionId: string;
  agentId: string;
  toolName: string;
  toolInput: any;
  status: "pending" | "approved" | "denied";
  createdAt: Date;
}

export interface ApprovalRule {
  toolPattern: RegExp;
  requireApproval: boolean;
  autoApprovePatterns?: RegExp[];
}

const DEFAULT_RULES: ApprovalRule[] = [
  {
    toolPattern: /^(Write|Edit)$/,
    requireApproval: true,
    autoApprovePatterns: [/^outputs\//], // Auto-approve writes to outputs/
  },
  {
    toolPattern: /^Bash$/,
    requireApproval: true,
    autoApprovePatterns: [/^(ls|cat|grep|find)/], // Auto-approve read-only commands
  },
  {
    toolPattern: /^(Read|Glob|Grep)$/,
    requireApproval: false,
  },
];

export class ApprovalManager extends EventEmitter {
  private pendingApprovals: Map<string, ApprovalRequest> = new Map();
  private approvalResolvers: Map<string, (approved: boolean) => void> =
    new Map();
  private rules: ApprovalRule[] = DEFAULT_RULES;

  async checkApproval(
    sessionId: string,
    agentId: string,
    toolName: string,
    toolInput: any
  ): Promise<boolean> {
    // Find matching rule
    const rule = this.rules.find((r) => r.toolPattern.test(toolName));

    if (!rule || !rule.requireApproval) {
      return true; // No approval needed
    }

    // Check auto-approve patterns
    const inputStr = JSON.stringify(toolInput);
    if (rule.autoApprovePatterns?.some((p) => p.test(inputStr))) {
      return true;
    }

    // Create approval request
    const request: ApprovalRequest = {
      id: crypto.randomUUID(),
      sessionId,
      agentId,
      toolName,
      toolInput,
      status: "pending",
      createdAt: new Date(),
    };

    this.pendingApprovals.set(request.id, request);
    this.emit("approvalRequired", request);

    // Save to database
    await db.run(
      `INSERT INTO approvals (id, session_id, tool_name, tool_input, status)
       VALUES (?, ?, ?, ?, ?)`,
      [request.id, sessionId, toolName, JSON.stringify(toolInput), "pending"]
    );

    // Wait for decision
    return new Promise((resolve) => {
      this.approvalResolvers.set(request.id, resolve);

      // Timeout after 5 minutes
      setTimeout(() => {
        if (this.approvalResolvers.has(request.id)) {
          this.approvalResolvers.delete(request.id);
          resolve(false);
        }
      }, 5 * 60 * 1000);
    });
  }

  async approve(approvalId: string): Promise<void> {
    await this.decide(approvalId, true);
  }

  async deny(approvalId: string): Promise<void> {
    await this.decide(approvalId, false);
  }

  private async decide(approvalId: string, approved: boolean): Promise<void> {
    const resolver = this.approvalResolvers.get(approvalId);
    if (resolver) {
      resolver(approved);
      this.approvalResolvers.delete(approvalId);
    }

    const request = this.pendingApprovals.get(approvalId);
    if (request) {
      request.status = approved ? "approved" : "denied";
      this.pendingApprovals.delete(approvalId);
    }

    await db.run(
      `UPDATE approvals SET status = ?, decided_at = ? WHERE id = ?`,
      [approved ? "approved" : "denied", new Date().toISOString(), approvalId]
    );

    this.emit("approvalDecided", { approvalId, approved });
  }

  getPendingApprovals(): ApprovalRequest[] {
    return Array.from(this.pendingApprovals.values());
  }
}
```

#### 3.2 Approval Hooks Integration
- [ ] Integrate approval manager with agent hooks
- [ ] Add approval callbacks to Claude SDK options

```typescript
// apps/server/src/hooks/approval.ts
import { ApprovalManager } from "../services/approval-manager";

export function createApprovalHooks(
  approvalManager: ApprovalManager,
  sessionId: string,
  agentId: string
) {
  return {
    PreToolUse: [
      {
        matcher: ".*",
        hooks: [
          {
            type: "callback" as const,
            callback: async (toolUse: any): Promise<boolean> => {
              const approved = await approvalManager.checkApproval(
                sessionId,
                agentId,
                toolUse.name,
                toolUse.input
              );
              return approved;
            },
          },
        ],
      },
    ],
  };
}
```

### Phase 4: Real-Time Communication (Week 4-5)

#### 4.1 WebSocket Server
- [ ] Set up WebSocket server
- [ ] Implement room-based subscriptions
- [ ] Add message broadcasting

```typescript
// apps/server/src/ws/handlers.ts
import { WebSocketServer, WebSocket } from "ws";
import { AgentManager } from "../services/agent-manager";
import { ApprovalManager } from "../services/approval-manager";

interface Client {
  ws: WebSocket;
  subscriptions: Set<string>;
}

export class WebSocketHandler {
  private clients: Map<string, Client> = new Map();
  private wss: WebSocketServer;

  constructor(
    private agentManager: AgentManager,
    private approvalManager: ApprovalManager
  ) {
    this.wss = new WebSocketServer({ noServer: true });
    this.setupEventForwarding();
  }

  handleUpgrade(request: any, socket: any, head: any): void {
    this.wss.handleUpgrade(request, socket, head, (ws) => {
      const clientId = crypto.randomUUID();
      this.clients.set(clientId, { ws, subscriptions: new Set() });

      ws.on("message", (data) => this.handleMessage(clientId, data));
      ws.on("close", () => this.clients.delete(clientId));

      ws.send(JSON.stringify({ type: "connected", clientId }));
    });
  }

  private handleMessage(clientId: string, data: any): void {
    const client = this.clients.get(clientId);
    if (!client) return;

    const message = JSON.parse(data.toString());

    switch (message.type) {
      case "subscribe":
        client.subscriptions.add(message.channel);
        break;

      case "unsubscribe":
        client.subscriptions.delete(message.channel);
        break;

      case "pauseAgent":
        this.agentManager.pauseAgent(message.agentId);
        break;

      case "resumeAgent":
        this.agentManager.resumeAgent(message.agentId, message.prompt);
        break;

      case "approve":
        this.approvalManager.approve(message.approvalId);
        break;

      case "deny":
        this.approvalManager.deny(message.approvalId);
        break;
    }
  }

  private setupEventForwarding(): void {
    // Forward agent events
    this.agentManager.on("agentStarted", (e) =>
      this.broadcast(`agent:${e.agentId}`, { type: "started", ...e })
    );

    this.agentManager.on("agentMessage", (e) =>
      this.broadcast(`agent:${e.agentId}`, { type: "message", ...e })
    );

    this.agentManager.on("agentToolUse", (e) =>
      this.broadcast(`agent:${e.agentId}`, { type: "toolUse", ...e })
    );

    this.agentManager.on("agentCompleted", (e) =>
      this.broadcast(`agent:${e.agentId}`, { type: "completed", ...e })
    );

    // Forward approval events
    this.approvalManager.on("approvalRequired", (e) =>
      this.broadcast("approvals", { type: "required", ...e })
    );

    this.approvalManager.on("approvalDecided", (e) =>
      this.broadcast("approvals", { type: "decided", ...e })
    );
  }

  private broadcast(channel: string, message: any): void {
    const payload = JSON.stringify({ channel, ...message });

    for (const client of this.clients.values()) {
      if (client.subscriptions.has(channel) || channel === "approvals") {
        client.ws.send(payload);
      }
    }
  }
}
```

#### 4.2 REST API
- [ ] Projects CRUD
- [ ] Agent control endpoints
- [ ] Approval endpoints

```typescript
// apps/server/src/api/agents.ts
import { Hono } from "hono";
import { AgentManager } from "../services/agent-manager";

export function createAgentRoutes(agentManager: AgentManager) {
  const app = new Hono();

  // Start workflow
  app.post("/projects/:projectId/workflow", async (c) => {
    const { projectId } = c.req.param();
    const body = await c.req.json();

    await agentManager.runWorkflow({
      projectId,
      projectPath: body.projectPath,
      prdContent: body.prdContent,
      agents: body.agents,
      parallel: body.parallel ?? true,
    });

    return c.json({ status: "started" });
  });

  // Pause agent
  app.post("/agents/:agentId/pause", (c) => {
    const { agentId } = c.req.param();
    agentManager.pauseAgent(agentId);
    return c.json({ status: "paused" });
  });

  // Resume agent
  app.post("/agents/:agentId/resume", async (c) => {
    const { agentId } = c.req.param();
    const body = await c.req.json();
    agentManager.resumeAgent(agentId, body.prompt);
    return c.json({ status: "resumed" });
  });

  // Inject message (step-in)
  app.post("/agents/:agentId/inject", async (c) => {
    const { agentId } = c.req.param();
    const body = await c.req.json();

    // Pause, inject, resume
    agentManager.pauseAgent(agentId);
    agentManager.resumeAgent(agentId, body.message);

    return c.json({ status: "injected" });
  });

  return app;
}
```

### Phase 5: Dashboard UI (Week 5-7)

#### 5.1 Core Components
- [ ] Agent status cards
- [ ] Real-time stream viewer
- [ ] Approval dialog
- [ ] Output viewer

#### 5.2 Pages
- [ ] Project list and creation
- [ ] Workflow configuration
- [ ] Agent monitoring dashboard
- [ ] Approval queue
- [ ] Output review

#### 5.3 Real-time Hooks
- [ ] WebSocket connection hook
- [ ] Agent stream subscription hook
- [ ] Approval notification hook

```typescript
// apps/dashboard/src/hooks/use-agent-stream.ts
import { useEffect, useState } from "react";
import { useWebSocket } from "./use-websocket";

interface AgentMessage {
  type: string;
  content: any;
  timestamp: string;
}

export function useAgentStream(agentId: string) {
  const [messages, setMessages] = useState<AgentMessage[]>([]);
  const [status, setStatus] = useState<string>("idle");
  const { subscribe, send } = useWebSocket();

  useEffect(() => {
    const unsubscribe = subscribe(`agent:${agentId}`, (message) => {
      switch (message.type) {
        case "started":
          setStatus("running");
          break;
        case "message":
          setMessages((prev) => [...prev, message.message]);
          break;
        case "completed":
          setStatus("completed");
          break;
        case "paused":
          setStatus("paused");
          break;
      }
    });

    return unsubscribe;
  }, [agentId, subscribe]);

  const pause = () => send({ type: "pauseAgent", agentId });
  const resume = (prompt?: string) =>
    send({ type: "resumeAgent", agentId, prompt });
  const inject = (message: string) => resume(message);

  return { messages, status, pause, resume, inject };
}
```

### Phase 6: Testing & Polish (Week 7-8)

#### 6.1 Testing
- [ ] Unit tests for core services
- [ ] Integration tests for agent workflows
- [ ] E2E tests for dashboard

#### 6.2 Documentation
- [ ] API documentation
- [ ] User guide
- [ ] Deployment guide

#### 6.3 Polish
- [ ] Error handling improvements
- [ ] Loading states
- [ ] Performance optimization

## Milestones

### Milestone 1: Single Agent MVP (End of Week 2)
- Run a single specialist agent from CLI
- Session persistence works
- Basic logging

### Milestone 2: Multi-Agent Orchestration (End of Week 4)
- All 5 specialist agents defined
- Parallel execution works
- Approval system functional

### Milestone 3: Dashboard Alpha (End of Week 6)
- Web dashboard shows agent status
- Real-time streaming works
- Can pause/resume agents

### Milestone 4: Production Ready (End of Week 8)
- Full approval workflow
- Output management
- Documentation complete

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Claude SDK API changes | Pin version, monitor changelog |
| Rate limiting | Implement retry with backoff, queue requests |
| Session state corruption | Regular snapshots, recovery mechanism |
| WebSocket disconnections | Auto-reconnect, message buffering |
| Large PRD handling | Chunking, summary generation |

## Success Criteria

1. **Functional**: All 5 specialist agents produce valid outputs from a sample PRD
2. **Performance**: 5 agents complete in parallel within 10 minutes
3. **Reliability**: No data loss on server restart
4. **Usability**: PM can complete full workflow via dashboard without CLI
5. **Observability**: All agent actions visible in real-time

## Next Steps After v1

1. **Custom agents**: Allow PM to define new specialist types
2. **Templates**: Pre-built PRD templates for common scenarios
3. **Integrations**: Jira, Linear, GitHub Issues
4. **Collaboration**: Multi-user support with role-based access
5. **Analytics**: Track agent performance, output quality metrics
