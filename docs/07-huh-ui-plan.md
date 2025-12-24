# Huh TUI Integration Plan

**Status**: Planned
**Library**: [charmbracelet/huh](https://github.com/charmbracelet/huh) v2

---

## Overview

Add interactive terminal UI forms to pagent using the `huh` library from Charm. This improves UX for configuration, agent selection, and runtime interactions.

## Goals

1. **Interactive `init`** - Guided setup wizard for first-time users
2. **Interactive `run`** - Agent selection and configuration when flags not provided
3. **Runtime prompts** - Confirmation dialogs, error recovery options
4. **Accessibility** - Screen reader support via huh's accessible mode

## Target Commands

| Command | Current | With Huh |
|---------|---------|----------|
| `pagent init` | Creates default config | Interactive form: persona, stack, preferences |
| `pagent run` | Requires flags or config | Interactive if no input specified |
| `pagent run <prd>` | Uses config/defaults | Optional `--interactive` for agent selection |

## Non-Goals

- Replace all CLI flags (flags remain for scripting/CI)
- Add huh to agent execution flow (keep non-interactive)
- Real-time TUI dashboard (out of scope for v1)

## User Experience

### `pagent init` Flow

```
┌─────────────────────────────────────────────────────┐
│ Welcome to Pagent Setup                             │
├─────────────────────────────────────────────────────┤
│                                                     │
│ Persona:  ○ minimal   ● balanced   ○ production    │
│                                                     │
│ Language: [ go                               ▼ ]   │
│                                                     │
│ Stack:                                              │
│   Cloud:    [ aws                            ▼ ]   │
│   Compute:  [ kubernetes                     ▼ ]   │
│   Database: [ postgres                       ▼ ]   │
│                                                     │
│ Agents:   [x] architect  [x] qa  [x] security      │
│           [x] implementer  [x] verifier            │
│                                                     │
│            [ Cancel ]  [ Create Config ]           │
└─────────────────────────────────────────────────────┘
```

### `pagent run` Flow (No Input)

```
┌─────────────────────────────────────────────────────┐
│ No input file specified                             │
├─────────────────────────────────────────────────────┤
│                                                     │
│ Input path: [ ./prd.md                        📁 ] │
│                                                     │
│ Or select recent:                                   │
│   > ./examples/sample-prd.md                       │
│     ./docs/requirements.md                         │
│                                                     │
│              [ Cancel ]  [ Run ]                   │
└─────────────────────────────────────────────────────┘
```

## Dependencies

```go
require github.com/charmbracelet/huh/v2 v2.x.x
```

## Success Criteria

- [ ] `pagent init` walks through config creation interactively
- [ ] `pagent run` without args prompts for input file
- [ ] `--no-interactive` flag disables all prompts
- [ ] `--accessible` flag enables screen reader compatible mode
- [ ] Auto-detect non-terminal environments and enable accessible mode
- [ ] All existing CLI flags continue to work

## Risks

| Risk | Mitigation |
|------|------------|
| Dependency bloat | huh is lightweight (~2MB) |
| Breaking CI scripts | Keep flags as primary, TUI as fallback |
| Terminal compatibility | Use huh's accessible mode detection |

## Timeline

1. **Phase 1**: Interactive `init` command
2. **Phase 2**: Interactive `run` (input selection)
3. **Phase 3**: Runtime confirmations and error recovery
