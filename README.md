# PM Agent Workflow

An AI-powered Product Manager workflow system built on the Claude Agent SDK. Orchestrate multiple specialist agents to transform PRDs into actionable deliverables.

## Overview

This system enables a Product Manager to:
- Upload a PRD (Product Requirements Document)
- Orchestrate specialist agents (Design, Tech, QA, Security, Infra)
- Monitor agents in real-time via dashboard
- Intervene ("step into") any agent session
- Approve/deny sensitive actions before execution

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Dashboard (Next.js)                       │
│   PRD Upload → Agent Monitor → Approvals → Output Review    │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│               Orchestration Server (Hono)                    │
│   Agent Manager → Approval Manager → Session Manager         │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                Claude Agent SDK Runtime                      │
│   Design Lead │ Tech Lead │ QA Lead │ Security │ Infra      │
└─────────────────────────────────────────────────────────────┘
```

## Specialist Agents

| Agent | Input | Output |
|-------|-------|--------|
| **Design Lead** | PRD | `design-spec.md` |
| **Tech Lead** | PRD, Design Spec | `technical-requirements.md` |
| **QA Lead** | PRD, Design Spec, TRD | `test-plan.md` |
| **Security Reviewer** | PRD, TRD | `security-assessment.md` |
| **Infra Lead** | PRD, TRD | `infrastructure-plan.md` |

## Key Features

### Real-Time Observation
- Live streaming of agent activity
- Tool call monitoring with inputs/outputs
- File access tracking

### Step-Into Capability
- Pause any running agent
- Inject messages/instructions
- Resume with modified context

### Approval Gates
- Configurable approval rules
- Auto-approve safe patterns
- Block dangerous operations until reviewed

### Session Persistence
- Survive server restarts
- Resume interrupted sessions
- Fork sessions for experimentation

## Tech Stack

- **Agent Runtime**: Claude Agent SDK (`@anthropic-ai/claude-agent-sdk`)
- **Backend**: Hono (TypeScript)
- **Frontend**: Next.js 14 + shadcn/ui
- **Database**: SQLite
- **Real-time**: WebSocket

## Quick Start

```bash
# Prerequisites
# - Node.js 20+
# - Claude Code installed
# - ANTHROPIC_API_KEY set

# Clone and install
git clone <repo-url>
cd pm-agent-workflow
pnpm install

# Start development servers
pnpm dev

# Open dashboard
open http://localhost:3000
```

## Project Structure

```
pm-agent-workflow/
├── apps/
│   ├── server/           # Orchestration server
│   └── dashboard/        # Next.js frontend
├── packages/
│   └── shared/           # Shared types
├── .claude/
│   ├── agents/           # Agent definitions
│   └── commands/         # Slash commands
└── docs/                 # Documentation
```

## Documentation

- [Research Summary](docs/01-research-summary.md) - Background research and findings
- [Framework Comparison](docs/02-framework-comparison.md) - Claude Agent SDK vs alternatives
- [Requirements](docs/03-requirements.md) - Full requirements specification
- [Implementation Plan](docs/04-implementation-plan.md) - Detailed implementation guide

## Usage

### 1. Create a Project

```bash
curl -X POST http://localhost:8080/api/projects \
  -H "Content-Type: application/json" \
  -d '{"name": "My Product", "prdPath": "./prd.md"}'
```

### 2. Start Workflow

```bash
curl -X POST http://localhost:8080/api/projects/{id}/workflow \
  -H "Content-Type: application/json" \
  -d '{"agents": ["design-lead", "tech-lead", "qa-lead"], "parallel": true}'
```

### 3. Monitor via Dashboard

Open `http://localhost:3000/projects/{id}` to:
- Watch agents work in real-time
- Approve/deny pending actions
- Step into any agent session

### 4. Review Outputs

All outputs saved to `outputs/` directory:
- `design-spec.md`
- `technical-requirements.md`
- `test-plan.md`
- `security-assessment.md`
- `infrastructure-plan.md`

## API Reference

### Projects

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/projects` | List all projects |
| POST | `/api/projects` | Create project |
| GET | `/api/projects/:id` | Get project details |

### Agents

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/projects/:id/workflow` | Start workflow |
| POST | `/api/agents/:id/pause` | Pause agent |
| POST | `/api/agents/:id/resume` | Resume agent |
| POST | `/api/agents/:id/inject` | Inject message |

### Approvals

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/approvals` | List pending approvals |
| POST | `/api/approvals/:id/approve` | Approve action |
| POST | `/api/approvals/:id/deny` | Deny action |

## WebSocket Events

Connect to `ws://localhost:8080/ws` and subscribe to channels:

```javascript
// Subscribe to agent events
ws.send(JSON.stringify({ type: "subscribe", channel: "agent:tech-lead" }));

// Subscribe to approvals
ws.send(JSON.stringify({ type: "subscribe", channel: "approvals" }));
```

### Event Types

- `started` - Agent began execution
- `message` - Agent produced output
- `toolUse` - Agent used a tool
- `completed` - Agent finished
- `approvalRequired` - Action needs approval

## Configuration

### Environment Variables

```bash
# Required
ANTHROPIC_API_KEY=sk-ant-...

# Optional
PORT=8080
DATABASE_URL=file:./data.db
LOG_LEVEL=info
```

### Approval Rules

Edit `config/approval-rules.json`:

```json
{
  "rules": [
    {
      "tool": "Write|Edit",
      "requireApproval": true,
      "autoApprovePatterns": ["^outputs/"]
    },
    {
      "tool": "Bash",
      "requireApproval": true,
      "autoApprovePatterns": ["^(ls|cat|grep)"]
    }
  ]
}
```

## Development

### Running Tests

```bash
pnpm test           # Run all tests
pnpm test:unit      # Unit tests only
pnpm test:e2e       # E2E tests
```

### Building

```bash
pnpm build          # Build all packages
pnpm build:server   # Build server only
pnpm build:dashboard # Build dashboard only
```

### Docker

```bash
docker-compose up -d    # Start all services
docker-compose logs -f  # View logs
```

## Roadmap

### v1.0 (Current)
- [x] Research and planning
- [ ] Core agent runtime
- [ ] Multi-agent orchestration
- [ ] Approval system
- [ ] Dashboard UI
- [ ] Session persistence

### v1.1
- [ ] Custom agent definitions
- [ ] PRD templates
- [ ] Output export (PDF, Confluence)

### v2.0
- [ ] Multi-user collaboration
- [ ] Jira/Linear integration
- [ ] Analytics dashboard
- [ ] Custom workflows

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests
5. Submit a pull request

## License

MIT

## Acknowledgments

- Built on [Claude Agent SDK](https://docs.claude.com/en/docs/agent-sdk/overview) by Anthropic
- Research informed by [Mastra](https://mastra.ai) and [VoltAgent](https://voltagent.dev)
# pagent
