package prompt

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/tuannvm/pm-agent-workflow/internal/types"
)

//go:embed templates/*.md
var embeddedTemplates embed.FS

// Type aliases for the shared types - used in prompt templates
type (
	TechStack               = types.TechStack
	ArchitecturePreferences = types.ArchitecturePreferences
	StackResolution         = types.StackResolution
	StackConflict           = types.StackConflict
)

// Variables holds the template variables for prompt rendering
type Variables struct {
	PRDPath       string   // Primary input file (backward compatible)
	InputFiles    []string // All input files (when input is a directory)
	InputDir      string   // Input directory path (empty if single file)
	HasMultiInput bool     // True if multiple input files
	OutputDir     string
	OutputPath    string
	AgentName     string
	ExistingFiles []string                // List of files already in OutputDir
	HasExisting   bool                    // True if there are existing outputs to consider
	Persona       string                  // Implementation style: minimal, balanced, production
	Stack         TechStack               // Technology stack preferences
	Preferences   ArchitecturePreferences // Architectural style preferences

	// Mode-specific variables for existing codebase modifications
	Mode           string // "create" or "modify"
	TargetCodebase string // Path to existing codebase (for modify mode)
	SpecsOutputDir string // Directory for spec outputs
	CodeOutputDir  string // Directory for code outputs

	// Resolution holds user-resolved conflicts from UI (nil if no UI interaction)
	Resolution *StackResolution

	// Custom allows arbitrary key-value pairs
	Custom map[string]string
}

// IsMinimal returns true if persona is "minimal"
func (v Variables) IsMinimal() bool {
	return v.Persona == "minimal"
}

// IsBalanced returns true if persona is "balanced"
func (v Variables) IsBalanced() bool {
	return v.Persona == "balanced"
}

// IsProduction returns true if persona is "production"
func (v Variables) IsProduction() bool {
	return v.Persona == "production"
}

// IsModifyMode returns true if mode is "modify"
func (v Variables) IsModifyMode() bool {
	return v.Mode == "modify"
}

// IsCreateMode returns true if mode is "create" or empty (default)
func (v Variables) IsCreateMode() bool {
	return v.Mode == "" || v.Mode == "create"
}

// HasTargetCodebase returns true if a target codebase is specified
func (v Variables) HasTargetCodebase() bool {
	return v.TargetCodebase != ""
}

// IsStateless returns true if stateless architecture is preferred
// This is true if either:
// 1. Preferences.Stateless is explicitly true, OR
// 2. All storage technologies (database, cache, message_queue) are "none" or empty
func (v Variables) IsStateless() bool {
	if v.Preferences.Stateless {
		return true
	}
	// Infer stateless if all storage is disabled
	return !v.HasDatabase() && !v.HasCache() && !v.HasMessageQueue()
}

// HasDatabase returns true if a database is configured (not "none" or empty)
func (v Variables) HasDatabase() bool {
	return v.Stack.Database != "" && v.Stack.Database != "none"
}

// HasCache returns true if a cache is configured (not "none" or empty)
func (v Variables) HasCache() bool {
	return v.Stack.Cache != "" && v.Stack.Cache != "none"
}

// HasMessageQueue returns true if a message queue is configured (not "none" or empty)
func (v Variables) HasMessageQueue() bool {
	return v.Stack.MessageQueue != "" && v.Stack.MessageQueue != "none"
}

// HasDataLake returns true if a data lake is configured (not "none" or empty)
func (v Variables) HasDataLake() bool {
	return v.Stack.DataLake != "" && v.Stack.DataLake != "none"
}

// IsGitHubActions returns true if compute is github-actions
func (v Variables) IsGitHubActions() bool {
	return v.Stack.Compute == "github-actions"
}

// IsKubernetes returns true if compute is kubernetes-based (kubernetes, eks, gke, aks)
func (v Variables) IsKubernetes() bool {
	switch v.Stack.Compute {
	case "kubernetes", "eks", "gke", "aks":
		return true
	default:
		return false
	}
}

// IsServerless returns true if compute is serverless (lambda, cloud-functions, etc.)
func (v Variables) IsServerless() bool {
	switch v.Stack.Compute {
	case "lambda", "cloud-functions", "azure-functions", "serverless":
		return true
	default:
		return false
	}
}

// NeedsContainerization returns true if the app should be containerized
// This is false for GitHub Actions or serverless compute
func (v Variables) NeedsContainerization() bool {
	if v.IsGitHubActions() || v.IsServerless() {
		return false
	}
	return v.Preferences.Containerized
}

// NeedsKubernetesManifests returns true if k8s manifests should be generated
func (v Variables) NeedsKubernetesManifests() bool {
	return v.IsKubernetes() && v.Preferences.IncludeIaC
}

// HasResolution returns true if UI-resolved conflicts are available
func (v Variables) HasResolution() bool {
	return v.Resolution != nil && v.Resolution.Resolved
}

// HasUnresolvedConflicts returns true if there are conflicts that need resolution
func (v Variables) HasUnresolvedConflicts() bool {
	return v.Resolution != nil && v.Resolution.HasConflicts()
}

// GetResolvedValue returns the resolved value for a category, or the config default
func (v Variables) GetResolvedValue(category string) string {
	if v.Resolution != nil {
		if resolved := v.Resolution.GetResolution(category); resolved != "" {
			return resolved
		}
	}
	// Fall back to config values
	switch category {
	case "database":
		return v.Stack.Database
	case "cache":
		return v.Stack.Cache
	case "message_queue":
		return v.Stack.MessageQueue
	case "compute":
		return v.Stack.Compute
	default:
		return ""
	}
}

// IsREST returns true if REST API style is preferred
func (v Variables) IsREST() bool {
	return v.Preferences.APIStyle == "" || v.Preferences.APIStyle == "rest"
}

// IsGraphQL returns true if GraphQL API style is preferred
func (v Variables) IsGraphQL() bool {
	return v.Preferences.APIStyle == "graphql"
}

// IsGRPC returns true if gRPC API style is preferred
func (v Variables) IsGRPC() bool {
	return v.Preferences.APIStyle == "grpc"
}

// WantsTests returns true if testing is enabled (any level)
func (v Variables) WantsTests() bool {
	return v.Preferences.TestingDepth != "none"
}

// WantsIntegrationTests returns true if integration tests are wanted
func (v Variables) WantsIntegrationTests() bool {
	return v.Preferences.TestingDepth == "integration" || v.Preferences.TestingDepth == "e2e"
}

// WantsE2ETests returns true if e2e tests are wanted
func (v Variables) WantsE2ETests() bool {
	return v.Preferences.TestingDepth == "e2e"
}

// WantsMinimalDeps returns true if minimal dependencies are preferred
func (v Variables) WantsMinimalDeps() bool {
	return v.Preferences.DependencyStyle == "minimal"
}

// WantsBatteries returns true if feature-rich dependencies are preferred
func (v Variables) WantsBatteries() bool {
	return v.Preferences.DependencyStyle == "batteries"
}

// WantsComprehensiveDocs returns true if comprehensive documentation is wanted
func (v Variables) WantsComprehensiveDocs() bool {
	return v.Preferences.DocumentationLevel == "comprehensive"
}

// WantsMinimalDocs returns true if minimal documentation is wanted
func (v Variables) WantsMinimalDocs() bool {
	return v.Preferences.DocumentationLevel == "minimal"
}

// Loader handles loading and rendering prompt templates
type Loader struct {
	promptsDir string
}

// NewLoader creates a new prompt loader
// promptsDir is the directory containing custom prompt templates (optional)
func NewLoader(promptsDir string) *Loader {
	return &Loader{
		promptsDir: promptsDir,
	}
}

// Load loads a prompt template for the given agent
// Priority order:
// 1. Inline prompt (if provided)
// 2. Custom prompt file (if promptFile is provided)
// 3. Prompt from promptsDir (if directory exists)
// 4. Embedded default template
func (l *Loader) Load(agentName, inlinePrompt, promptFile string) (string, error) {
	// Priority 1: Inline prompt
	if inlinePrompt != "" {
		return inlinePrompt, nil
	}

	// Priority 2: Custom prompt file
	if promptFile != "" {
		content, err := os.ReadFile(promptFile)
		if err != nil {
			return "", fmt.Errorf("failed to read prompt file %s: %w", promptFile, err)
		}
		return string(content), nil
	}

	// Priority 3: Prompt from promptsDir
	if l.promptsDir != "" {
		promptPath := filepath.Join(l.promptsDir, agentName+".md")
		if content, err := os.ReadFile(promptPath); err == nil {
			return string(content), nil
		}
		// File doesn't exist, fall through to embedded
	}

	// Priority 4: Embedded default template
	content, err := embeddedTemplates.ReadFile("templates/" + agentName + ".md")
	if err != nil {
		return "", fmt.Errorf("no prompt template found for agent %s", agentName)
	}
	return string(content), nil
}

// Render renders a prompt template with the given variables
func (l *Loader) Render(promptTemplate string, vars Variables) (string, error) {
	// Convert old-style placeholders to Go template syntax
	prompt := convertLegacyPlaceholders(promptTemplate)

	// Create template with custom functions
	// Use missingkey=error to catch typos in template variables
	tmpl, err := template.New("prompt").
		Option("missingkey=error").
		Funcs(template.FuncMap{
			"join":  strings.Join,
			"upper": strings.ToUpper,
			"lower": strings.ToLower,
		}).Parse(prompt)
	if err != nil {
		return "", fmt.Errorf("failed to parse prompt template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("failed to render prompt template: %w", err)
	}

	return buf.String(), nil
}

// LoadAndRender loads and renders a prompt in one step
func (l *Loader) LoadAndRender(agentName, inlinePrompt, promptFile string, vars Variables) (string, error) {
	tmpl, err := l.Load(agentName, inlinePrompt, promptFile)
	if err != nil {
		return "", err
	}
	return l.Render(tmpl, vars)
}

// convertLegacyPlaceholders converts old {placeholder} style to {{.Field}} style
func convertLegacyPlaceholders(prompt string) string {
	replacements := map[string]string{
		"{prd_path}":    "{{.PRDPath}}",
		"{output_path}": "{{.OutputPath}}",
		"{output_dir}":  "{{.OutputDir}}",
		"{agent_name}":  "{{.AgentName}}",
	}

	result := prompt
	for old, new := range replacements {
		result = strings.ReplaceAll(result, old, new)
	}
	return result
}

// ListAvailable returns a list of available prompt templates
func (l *Loader) ListAvailable() ([]string, error) {
	agents := make(map[string]bool)

	// Check embedded templates
	entries, err := embeddedTemplates.ReadDir("templates")
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
				name := strings.TrimSuffix(entry.Name(), ".md")
				agents[name] = true
			}
		}
	}

	// Check promptsDir
	if l.promptsDir != "" {
		entries, err := os.ReadDir(l.promptsDir)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
					name := strings.TrimSuffix(entry.Name(), ".md")
					agents[name] = true
				}
			}
		}
	}

	result := make([]string, 0, len(agents))
	for name := range agents {
		result = append(result, name)
	}
	return result, nil
}
