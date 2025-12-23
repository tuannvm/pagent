package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadEmbeddedTemplate(t *testing.T) {
	loader := NewLoader("") // No custom dir, use embedded only

	prompt, err := loader.Load("architect", "", "")
	if err != nil {
		t.Fatalf("Failed to load embedded architect template: %v", err)
	}

	if !strings.Contains(prompt, "Principal Software Architect") {
		t.Errorf("Embedded template doesn't contain expected content")
	}
}

func TestLoadCustomPromptFile(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	customPrompt := "Custom prompt for {{.AgentName}}"
	promptFile := filepath.Join(tmpDir, "custom.md")
	if err := os.WriteFile(promptFile, []byte(customPrompt), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader("")
	prompt, err := loader.Load("test", "", promptFile)
	if err != nil {
		t.Fatalf("Failed to load custom prompt file: %v", err)
	}

	if prompt != customPrompt {
		t.Errorf("Expected %q, got %q", customPrompt, prompt)
	}
}

func TestLoadInlinePromptTakesPrecedence(t *testing.T) {
	loader := NewLoader("")
	inlinePrompt := "Inline prompt takes precedence"

	prompt, err := loader.Load("architect", inlinePrompt, "")
	if err != nil {
		t.Fatal(err)
	}

	if prompt != inlinePrompt {
		t.Errorf("Expected inline prompt to take precedence")
	}
}

func TestRenderTemplate(t *testing.T) {
	loader := NewLoader("")

	vars := Variables{
		PRDPath:    "/path/to/prd.md",
		OutputDir:  "/path/to/outputs",
		OutputPath: "/path/to/outputs/architecture.md",
		AgentName:  "architect",
	}

	template := "Read PRD at {{.PRDPath}} and write to {{.OutputPath}}"
	result, err := loader.Render(template, vars)
	if err != nil {
		t.Fatal(err)
	}

	expected := "Read PRD at /path/to/prd.md and write to /path/to/outputs/architecture.md"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestRenderLegacyPlaceholders(t *testing.T) {
	loader := NewLoader("")

	vars := Variables{
		PRDPath:    "/path/to/prd.md",
		OutputPath: "/path/to/output.md",
	}

	// Old-style placeholders
	template := "Read {prd_path} and write to {output_path}"
	result, err := loader.Render(template, vars)
	if err != nil {
		t.Fatal(err)
	}

	expected := "Read /path/to/prd.md and write to /path/to/output.md"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestRenderMissingKeyErrors(t *testing.T) {
	loader := NewLoader("")

	vars := Variables{
		PRDPath: "/path/to/prd.md",
	}

	// Template with typo - should error
	template := "{{.PRDPaths}}" // Note: typo 'PRDPaths' instead of 'PRDPath'
	_, err := loader.Render(template, vars)
	if err == nil {
		t.Error("Expected error for missing key, got nil")
	}
}

func TestLoadFromPromptsDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a custom prompt in the prompts dir
	customContent := "Custom architect from prompts dir"
	if err := os.WriteFile(filepath.Join(tmpDir, "architect.md"), []byte(customContent), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(tmpDir)
	prompt, err := loader.Load("architect", "", "")
	if err != nil {
		t.Fatal(err)
	}

	if prompt != customContent {
		t.Errorf("Expected custom content from prompts dir, got embedded template")
	}
}

func TestLoadPriorityOrder(t *testing.T) {
	tmpDir := t.TempDir()
	promptsDir := filepath.Join(tmpDir, "prompts")
	os.MkdirAll(promptsDir, 0755)

	// Create prompts in prompts dir
	promptsDirContent := "From prompts dir"
	os.WriteFile(filepath.Join(promptsDir, "architect.md"), []byte(promptsDirContent), 0644)

	// Create custom file
	customFile := filepath.Join(tmpDir, "custom.md")
	customFileContent := "From custom file"
	os.WriteFile(customFile, []byte(customFileContent), 0644)

	inlineContent := "From inline"

	loader := NewLoader(promptsDir)

	// Test 1: Inline takes precedence over all
	result, _ := loader.Load("architect", inlineContent, customFile)
	if result != inlineContent {
		t.Errorf("Test 1 failed: inline should take precedence, got %q", result)
	}

	// Test 2: Custom file takes precedence over promptsDir
	result, _ = loader.Load("architect", "", customFile)
	if result != customFileContent {
		t.Errorf("Test 2 failed: custom file should take precedence, got %q", result)
	}

	// Test 3: promptsDir takes precedence over embedded
	result, _ = loader.Load("architect", "", "")
	if result != promptsDirContent {
		t.Errorf("Test 3 failed: promptsDir should take precedence over embedded, got %q", result)
	}
}
