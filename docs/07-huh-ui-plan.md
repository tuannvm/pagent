# Huh TUI Integration Plan

**Status**: Planned
**Library**: [charmbracelet/huh](https://github.com/charmbracelet/huh) v2

---

## Overview

Add an explicit `pagent ui` command that provides a single-screen interactive dashboard for running agents. Inspired by [gum](https://github.com/charmbracelet/gum)'s clean aesthetic - intuitive for new users, powerful for experts who don't want to memorize flags.

## Design Philosophy

1. **Explicit over implicit** - TUI via dedicated command, not magic fallback behavior
2. **Single screen** - All options visible at once, no multi-step wizards
3. **Smart defaults** - Pre-filled from config, ready to run immediately
4. **Progressive disclosure** - Advanced options collapsed by default
5. **Zero memorization** - See all options, tweak what you need, hit Enter

## Command Structure

```bash
pagent ui                    # Launch interactive dashboard
pagent ui ./custom.md        # Pre-fill input file
pagent ui --accessible       # Screen reader mode

# Existing commands unchanged (non-interactive)
pagent init                  # Creates default config
pagent run ./prd.md          # Runs directly with flags
```

## Non-Goals

- Replace existing CLI commands (they remain for scripting/CI)
- Multi-step wizard flows (single screen only)
- Interactive prompts during agent execution
- Real-time TUI dashboard showing agent progress

## User Interface

### Main Screen

```
┌─────────────────────────────────────────────────────────────┐
│  pagent                                                     │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Input     ./examples/sample-prd.md                    📁  │
│                                                             │
│  Agents    ● all  ○ select...                              │
│                                                             │
│  Persona   ○ minimal  ● balanced  ○ production             │
│                                                             │
│  Output    ./outputs                                        │
│                                                             │
│  ▶ Advanced                                                 │
│                                                             │
│                              [ Cancel ]  [ Run ↵ ]         │
└─────────────────────────────────────────────────────────────┘
```

### Agent Selection (when "select..." chosen)

```
  Agents    ☑ architect  ☑ implementer  ☑ qa
            ☑ security   ☑ verifier
```

### Advanced Options (expanded)

```
  ▼ Advanced
    Execution     ○ parallel  ○ sequential
    Resume        ○ normal  ○ skip up-to-date  ○ force regenerate
    Architecture  ○ stateless  ○ database-backed  ○ from config
    Timeout       0 (unlimited)
    Config        .pagent/config.yaml                         📁
    Verbosity     ○ normal  ○ verbose  ○ quiet
```

### Modify Mode (when target_codebase configured)

```
  ▼ Modify Mode
    Target        ./my-project
    Specs output  ./my-project/.pagent/specs
    Code output   ./my-project
```

## Option Coverage

All `pagent run` flags are available in the UI:

| Flag | UI Location |
|------|-------------|
| `<input>` | Input field with file picker |
| `-a/--agents` | Agents selector |
| `-p/--persona` | Persona radio buttons |
| `-o/--output` | Output field |
| `-s/--sequential` | Advanced → Execution |
| `-r/--resume` | Advanced → Resume (skip up-to-date) |
| `-f/--force` | Advanced → Resume (force regenerate) |
| `--stateless` | Advanced → Architecture |
| `--no-stateless` | Advanced → Architecture |
| `-t/--timeout` | Advanced → Timeout |
| `-c/--config` | Advanced → Config |
| `-v/--verbose` | Advanced → Verbosity |
| `-q/--quiet` | Advanced → Verbosity |

## Smart Defaults

1. **Input file**: Auto-detect `.md`, `.yaml` files in current directory; show recent files first
2. **Agents**: Default to "all" or from config's `enabled_agents`
3. **Persona**: From config or "balanced"
4. **Output**: From config or "./outputs"
5. **Advanced**: Collapsed, uses config values

## Dependencies

```go
require (
    github.com/charmbracelet/huh/v2 v2.x.x
    github.com/charmbracelet/lipgloss v1.x.x  // for gum-like styling
    golang.org/x/term v0.x.x                   // terminal detection
)
```

## Success Criteria

- [ ] `pagent ui` launches single-screen dashboard
- [ ] All `pagent run` flags accessible via UI
- [ ] Smart file discovery (recent files, glob patterns)
- [ ] Pre-populated from `.pagent/config.yaml`
- [ ] `--accessible` flag for screen reader mode
- [ ] Auto-detect non-terminal and enable accessible mode
- [ ] Enter key runs with current selections
- [ ] Escape/Cancel exits without running

## Risks

| Risk | Mitigation |
|------|------------|
| Dependency bloat | huh is lightweight (~2MB) |
| Terminal compatibility | Use huh's accessible mode detection |
| Complex form state | Single screen limits complexity |

## Phases

1. **Phase 1**: Core UI with input, agents, persona, output
2. **Phase 2**: Advanced options panel
3. **Phase 3**: Modify mode support, file picker improvements
