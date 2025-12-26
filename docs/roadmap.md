# Roadmap

> **pagent** orchestrates multiple AI coding agents (Claude Code, Gemini CLI, Codex) to work on software projects using structured PM workflows.

## v0.x (Current)

| Area | Status |
|------|--------|
| LLM Support | Claude Code only (AgentAPI) |
| Config | Full-featured, complex |
| Interface | CLI + TUI dashboard |

## v1.x (Next)

### P0: Multi-LLM Support

| Provider | Backend | Status |
|----------|---------|--------|
| Claude Code | Claude | ✅ Supported |
| Gemini CLI | Gemini | 🔄 In progress |
| Codex CLI | OpenAI | 📋 Backlog |
| AMP | Sourcegraph | 📋 Backlog |

**Why this matters:**
- Per-agent LLM selection (use what works best for each task)
- Mix LLMs in workflows (Claude for design, Codex for implementation)
- No hard dependency on Claude Code installation

### P1: Simplified Configuration

Current config requires too many decisions upfront.

**Changes:**
- Reduce required fields to 3: `task`, `repo`, `llm`
- Smart defaults that work for 80% of cases
- Preset profiles: `--preset go-api`, `--preset python-ml`
- Progressive disclosure for advanced options

### P2: UX Polish

- Fewer CLI flags (merge redundant options)
- Actionable error messages with fix suggestions
- Interactive setup: `pagent init`
- Concise agent output summaries

## v2.x (Future)

| Feature | Description |
|---------|-------------|
| Plugin system | Custom agents via Go plugins or external binaries |
| Cost tracking | Token usage, estimated cost per run, budgets |
| IDE extensions | VS Code, JetBrains integration |
| Team mode | Shared configs, agent templates, audit logs |
