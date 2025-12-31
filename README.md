# Pagent Marketplace

**Transform Product Requirements Documents (PRDs) into working software through 5 specialized AI agents.**

This is a Claude Code plugin marketplace containing the **pagent** plugin.

## Quick Start

### Installation

```bash
# Clone this repository
git clone https://github.com/tuannvm/pagent.git
cd pagent

# Add as a local marketplace
claude plugin marketplace add $(pwd)

# Install the plugin
claude plugin install pagent@pagent
```

### Usage

Once installed, run from any Claude Code session:

```bash
# Start a pipeline with your PRD
/pagent-run ./your-prd.md

# Check progress
/pagent-status

# Cancel if needed
/pagent-cancel
```

## Plugins

### [pagent](./plugins/pagent/)

Transform PRDs into architecture, test plans, security assessments, production-ready code, and verification reports through 5 specialized AI agents:

| Stage | Agent | Output |
|-------|-------|--------|
| 1 | architect | `architecture.md` |
| 2 | qa | `test-plan.md` |
| 2 | security | `security-assessment.md` |
| 3 | implementer | `code/` |
| 4 | verifier | `verification-report.md` |

## Marketplace Structure

```
pagent/
├── .claude-plugin/
│   └── marketplace.json      # Marketplace definition
├── plugins/
│   └── pagent/               # Pagent plugin
│       ├── .claude-plugin/
│       │   └── plugin.json
│       ├── commands/
│       ├── hooks/
│       ├── scripts/
│       └── README.md
├── docs/                     # Documentation
└── README.md
```

## Documentation

| Doc | Content |
|-----|---------|
| [Tutorial](docs/tutorial.md) | Step-by-step usage guide |
| [Architecture](docs/architecture.md) | Technical design and internals |
| [Roadmap](docs/roadmap.md) | Future plans |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT
