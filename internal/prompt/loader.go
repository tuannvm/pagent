package prompt

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed templates/*.md
var embeddedTemplates embed.FS

// Variables holds the template variables for prompt rendering
type Variables struct {
	PRDPath   string
	OutputDir string
	OutputPath string
	AgentName string
	// Custom allows arbitrary key-value pairs
	Custom map[string]string
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
			"join": strings.Join,
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
