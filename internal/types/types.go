// Package types contains shared type definitions used across the pm-agents codebase.
// This avoids duplication between config and prompt packages.
package types

// TechStack defines the technology preferences for code generation.
// Used by both configuration loading and prompt rendering.
type TechStack struct {
	// Cloud provider and compute
	Cloud   string `yaml:"cloud"`   // aws, gcp, azure
	Compute string `yaml:"compute"` // eks, gke, aks, ec2, lambda, github-actions, none

	// Databases
	// Use "none" or empty string to indicate no database (stateless architecture)
	Database string `yaml:"database"` // postgres, mongodb, mysql, none
	Cache    string `yaml:"cache"`    // redis, memcached, kvrock, none
	Search   string `yaml:"search"`   // elasticsearch, opensearch, none

	// Messaging and streaming
	// Use "none" or empty string to indicate synchronous/no message queue
	MessageQueue string `yaml:"message_queue"` // kafka, sqs, rabbitmq, nats, none

	// Infrastructure automation
	IaC    string `yaml:"iac"`    // terraform, pulumi, cloudformation
	GitOps string `yaml:"gitops"` // argocd, flux
	CI     string `yaml:"ci"`     // github-actions, gitlab-ci, jenkins

	// Data platform
	DataLake    string `yaml:"data_lake"`    // s3, gcs, adls
	DataEngine  string `yaml:"data_engine"`  // spark, flink
	QueryEngine string `yaml:"query_engine"` // trino, presto, athena

	// Observability
	Monitoring string `yaml:"monitoring"` // grafana, datadog, newrelic
	Alerting   string `yaml:"alerting"`   // pagerduty, opsgenie
	Logging    string `yaml:"logging"`    // loki, elasticsearch, cloudwatch

	// Communication
	Chat string `yaml:"chat"` // slack, teams

	// Custom/additional tools (free-form)
	Additional []string `yaml:"additional"`
}

// ArchitecturePreferences defines architectural style preferences.
// All fields have sensible defaults but can be customized per user/project needs.
type ArchitecturePreferences struct {
	// Stateless prefers stateless architectures over traditional databases
	// When true: Use event-driven patterns, external state stores (Redis, S3),
	// message queues for state propagation, idempotent operations
	// When false: Use traditional database-backed CRUD patterns
	// Default: false (traditional database-backed)
	Stateless bool `yaml:"stateless"`

	// APIStyle defines the API paradigm
	// Options: rest, graphql, grpc
	// Default: rest
	APIStyle string `yaml:"api_style"`

	// Language is the primary programming language
	// Options: go, python, typescript, java, rust
	// Default: go
	Language string `yaml:"language"`

	// TestingDepth controls how much testing code to generate
	// Options: none, unit, integration, e2e (includes all lower levels)
	// Default: unit
	TestingDepth string `yaml:"testing_depth"`

	// DocumentationLevel controls documentation verbosity
	// Options: minimal, standard, comprehensive
	// Default: standard
	DocumentationLevel string `yaml:"documentation_level"`

	// DependencyStyle controls third-party dependency usage
	// Options:
	//   - minimal: Prefer Go stdlib (net/http, encoding/json, database/sql)
	//   - standard: Common well-maintained libs where they add value
	//   - batteries: Feature-rich libs for faster development
	// Default: minimal
	DependencyStyle string `yaml:"dependency_style"`

	// ErrorHandling controls error handling sophistication
	// Options: simple, structured, comprehensive
	// Default: structured
	ErrorHandling string `yaml:"error_handling"`

	// Containerized indicates if the app should be containerized
	// Default: true
	Containerized bool `yaml:"containerized"`

	// IncludeCI indicates if CI/CD pipelines should be generated
	// Default: true
	IncludeCI bool `yaml:"include_ci"`

	// IncludeIaC indicates if infrastructure-as-code should be generated
	// Default: true
	IncludeIaC bool `yaml:"include_iac"`
}

// DefaultStack returns the default technology stack preferences.
// These are widely-used, well-documented technologies.
func DefaultStack() TechStack {
	return TechStack{
		// Cloud and compute
		Cloud:   "aws",
		Compute: "kubernetes", // Generic K8s (user can specify eks/gke/aks)

		// Databases
		Database: "postgres", // Most widely used, well-documented
		Cache:    "redis",

		// Messaging (empty = not needed unless specified)
		MessageQueue: "",

		// Infrastructure automation
		IaC:    "terraform",
		GitOps: "argocd",
		CI:     "github-actions",

		// Data platform (empty = not needed unless specified)
		DataLake:    "",
		DataEngine:  "",
		QueryEngine: "",

		// Observability
		Monitoring: "prometheus",
		Alerting:   "",
		Logging:    "stdout", // Structured JSON to stdout (K8s standard)

		// Communication (empty = not needed unless specified)
		Chat: "",

		// Additional tools (empty by default)
		Additional: []string{},
	}
}

// StackConflict represents a detected conflict between PRD and config
type StackConflict struct {
	Category    string // "database", "compute", "cache", "message_queue"
	ConfigValue string // What the config specifies
	PRDHint     string // What the PRD suggests (extracted keyword)
	Resolved    bool   // Whether this conflict has been resolved
	Resolution  string // The resolved value (if Resolved is true)
}

// StackResolution holds user-resolved conflicts from the UI
// This is populated by the interactive UI before agents run
type StackResolution struct {
	// Resolved indicates that the user has reviewed and resolved conflicts
	Resolved bool `yaml:"resolved"`

	// Conflicts lists all detected conflicts and their resolutions
	Conflicts []StackConflict `yaml:"conflicts,omitempty"`

	// EffectiveStack is the final stack after applying resolutions
	// This takes precedence over Config.Stack when provided
	EffectiveStack *TechStack `yaml:"effective_stack,omitempty"`
}

// HasConflicts returns true if there are unresolved conflicts
func (r *StackResolution) HasConflicts() bool {
	if r == nil {
		return false
	}
	for _, c := range r.Conflicts {
		if !c.Resolved {
			return true
		}
	}
	return false
}

// GetResolution returns the resolution for a specific category, or empty if not found
func (r *StackResolution) GetResolution(category string) string {
	if r == nil {
		return ""
	}
	for _, c := range r.Conflicts {
		if c.Category == category && c.Resolved {
			return c.Resolution
		}
	}
	return ""
}

// DefaultPreferences returns the default architecture preferences.
// These are neutral defaults that work for most projects.
func DefaultPreferences() ArchitecturePreferences {
	return ArchitecturePreferences{
		Stateless:          false,      // Traditional database-backed (most common)
		APIStyle:           "rest",     // REST is most widely understood
		Language:           "go",       // Go is the default language
		TestingDepth:       "unit",     // Unit tests as baseline
		DocumentationLevel: "standard", // README + basic docs
		DependencyStyle:    "minimal",  // Prefer stdlib over third-party packages
		ErrorHandling:      "structured", // Typed errors with context
		Containerized:      true,       // Docker/K8s is standard
		IncludeCI:          true,       // CI/CD is expected
		IncludeIaC:         true,       // IaC for reproducibility
	}
}
