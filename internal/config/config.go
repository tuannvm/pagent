package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// Config represents the pm-agents configuration
type Config struct {
	OutputDir string                 `yaml:"output_dir"`
	Timeout   int                    `yaml:"timeout"`
	Agents    map[string]AgentConfig `yaml:"agents"`
}

// AgentConfig represents a single agent's configuration
type AgentConfig struct {
	Prompt    string   `yaml:"prompt"`
	Output    string   `yaml:"output"`
	DependsOn []string `yaml:"depends_on"`
}

// Load reads config from file, checking multiple locations
func Load(path string) (*Config, error) {
	var configPath string

	if path != "" {
		configPath = path
	} else {
		// Check standard locations
		locations := []string{
			".pm-agents/config.yaml",
			".pm-agents/config.yml",
			filepath.Join(os.Getenv("HOME"), ".pm-agents/config.yaml"),
		}

		for _, loc := range locations {
			if _, err := os.Stat(loc); err == nil {
				configPath = loc
				break
			}
		}
	}

	if configPath == "" {
		return nil, os.ErrNotExist
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Apply defaults
	if cfg.OutputDir == "" {
		cfg.OutputDir = "./outputs"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 300
	}

	// Apply environment variable overrides
	cfg.ApplyEnvOverrides()

	return &cfg, nil
}

// ApplyEnvOverrides applies environment variable overrides to config
func (c *Config) ApplyEnvOverrides() {
	if envDir := os.Getenv("PM_AGENTS_OUTPUT_DIR"); envDir != "" {
		c.OutputDir = envDir
	}
	if envTimeout := os.Getenv("PM_AGENTS_TIMEOUT"); envTimeout != "" {
		var timeout int
		if _, err := fmt.Sscanf(envTimeout, "%d", &timeout); err == nil && timeout > 0 {
			c.Timeout = timeout
		}
	}
}

// Default returns the default configuration
func Default() *Config {
	return &Config{
		OutputDir: "./outputs",
		Timeout:   0, // 0 = no timeout (poll until completion). Set via --timeout for safety net.
		Agents: map[string]AgentConfig{
			// ============================================================
			// SPECIFICATION PHASE - Produces design documents
			// ============================================================
			"architect": {
				Prompt: `You are a Principal Software Architect. Read the PRD at {prd_path} and create a comprehensive architecture document.
Write your output to {output_path}.

This is THE source of truth for all technical decisions. Include:

## 1. System Overview
- High-level architecture diagram (mermaid)
- Component responsibilities
- Technology stack decisions with rationale

## 2. API Design
- RESTful API endpoints (OpenAPI 3.0 format)
- Request/response schemas
- Authentication flow
- Error response format

## 3. Data Models
- Entity relationship diagram (mermaid)
- Complete schema definitions with field types
- Database table designs (PostgreSQL)
- Indexes and constraints

## 4. Component Design
- UI/UX requirements and user flows
- Service layer responsibilities
- Repository layer patterns

## 5. Infrastructure
- Deployment architecture
- Scaling considerations
- Monitoring requirements

Be specific and actionable. This document drives all implementation.`,
				Output:    "architecture.md",
				DependsOn: []string{},
			},
			"qa": {
				Prompt: `You are a QA Lead. Read the PRD at {prd_path} and architecture at outputs/architecture.md.
Write your output to {output_path}.

Include:
- Test strategy overview
- Test cases organized by feature (mapped to API endpoints)
- Acceptance criteria for each user story
- Edge cases and error scenarios
- Performance test requirements
- Integration test scenarios

Be thorough and cover both happy paths and edge cases.`,
				Output:    "test-plan.md",
				DependsOn: []string{"architect"},
			},
			"security": {
				Prompt: `You are a Security Reviewer. Read the PRD at {prd_path} and architecture at outputs/architecture.md.
Write your output to {output_path}.

Include:
- Threat model (STRIDE analysis)
- Security requirements checklist
- Authentication/Authorization review
- Data protection requirements (encryption, PII handling)
- API security (rate limiting, input validation)
- Risk assessment with severity levels
- Required mitigations (must be addressed by implementer)

Focus on practical, actionable security guidance that the implementer MUST follow.`,
				Output:    "security-assessment.md",
				DependsOn: []string{"architect"},
			},

			// ============================================================
			// IMPLEMENTATION PHASE - Produces working code
			// ============================================================
			"implementer": {
				Prompt: `You are a Senior Full-Stack Go Developer. Your task is to implement the COMPLETE application.

Read these inputs carefully:
- PRD: {prd_path}
- Architecture: outputs/architecture.md (THIS IS YOUR SOURCE OF TRUTH)
- Security: outputs/security-assessment.md (MUST address all security requirements)

IMPORTANT: You own ALL code. Create a cohesive, working codebase.

Create this structure in outputs/code/:

## Database Layer
migrations/
├── 000001_create_users.up.sql
├── 000001_create_users.down.sql
├── 000002_create_*.up.sql (as needed)
└── 000002_create_*.down.sql
internal/db/
├── db.go                    # Connection pool, transactions
└── queries.sql              # SQL queries for sqlc
sqlc.yaml                    # sqlc configuration

## Application Layer
cmd/server/main.go           # Entry point, DI setup
internal/
├── config/config.go         # Configuration
├── model/models.go          # Domain models (match architecture.md exactly)
├── repository/              # Database operations (one per entity)
├── service/                 # Business logic
├── handler/                 # HTTP handlers
└── middleware/              # Auth, logging, error handling

## Build & Deploy
go.mod, go.sum
Makefile
Dockerfile
README.md

Requirements:
- Follow architecture.md EXACTLY for API endpoints and data models
- Implement ALL security mitigations from security-assessment.md
- Use Chi for HTTP routing
- Use pgx for PostgreSQL
- JWT authentication with refresh tokens
- Structured logging with zerolog
- Input validation with go-playground/validator
- Proper error handling and HTTP status codes

Write completion marker to {output_path} when done.`,
				Output:    "code/.complete",
				DependsOn: []string{"architect", "security"},
			},
			"verifier": {
				Prompt: `You are a Code Reviewer and Test Engineer. Your task is to verify the implementation and write tests.

Read these inputs:
- PRD: {prd_path}
- Architecture: outputs/architecture.md
- Security: outputs/security-assessment.md
- Test Plan: outputs/test-plan.md
- Implemented Code: outputs/code/

## Task 1: Verification Report
Create outputs/verification-report.md with:

### API Compliance
- [ ] All endpoints from architecture.md implemented
- [ ] Request/response schemas match specification
- [ ] Error responses follow defined format

### Security Compliance
- [ ] All mitigations from security-assessment.md addressed
- [ ] Authentication implemented correctly
- [ ] Input validation present
- [ ] No hardcoded secrets

### Code Quality
- [ ] Code compiles (go build ./...)
- [ ] No obvious bugs or logic errors
- [ ] Proper error handling
- [ ] Follows Go idioms

List any discrepancies found.

## Task 2: Write Tests
Create test files in outputs/code/:
internal/handler/*_test.go
internal/service/*_test.go
internal/repository/*_test.go
internal/testutil/
├── fixtures.go
└── mocks.go

Requirements:
- Table-driven tests
- Test happy paths and error cases
- Mock database for unit tests
- Use testify for assertions

Write completion marker to {output_path} when done.`,
				Output:    "code/.verified",
				DependsOn: []string{"implementer", "qa"},
			},
		},
	}
}

// GetAgentNames returns sorted list of agent names
func (c *Config) GetAgentNames() []string {
	names := make([]string, 0, len(c.Agents))
	for name := range c.Agents {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetDependencies returns the dependencies for an agent
func (c *Config) GetDependencies(agentName string) []string {
	if agent, ok := c.Agents[agentName]; ok {
		return agent.DependsOn
	}
	return nil
}
