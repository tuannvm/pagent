package postprocess

import (
	"strings"
	"testing"

	"github.com/tuannvm/pm-agent-workflow/internal/config"
)

func TestExtractSummary(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		contains []string
		excludes []string
	}{
		{
			name: "extracts first section",
			content: `# Architecture Document

This is the summary section.
It has multiple lines.

## Next Section

This should not be included.`,
			contains: []string{"This is the summary section", "multiple lines"},
			excludes: []string{"This should not be included", "## Next Section"},
		},
		{
			name: "handles h2 as first heading",
			content: `## Summary

The summary content here.

## Another Section

Not included.`,
			contains: []string{"The summary content here"},
			excludes: []string{"Not included"},
		},
		{
			name: "empty content returns empty",
			content:  "",
			contains: []string{},
			excludes: []string{},
		},
		{
			name:     "no headers returns empty",
			content:  "Just some text without headers",
			contains: []string{},
			excludes: []string{"Just some text"},
		},
		{
			name: "respects 20 line limit",
			content: `# Title

Line 1
Line 2
Line 3
Line 4
Line 5
Line 6
Line 7
Line 8
Line 9
Line 10
Line 11
Line 12
Line 13
Line 14
Line 15
Line 16
Line 17
Line 18
Line 19
Line 20
Line 21
Line 22

## Next Section`,
			contains: []string{"Line 1", "Line 20", "..."},
			excludes: []string{"Line 22"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSummary(tt.content)

			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("extractSummary() should contain %q, got:\n%s", s, result)
				}
			}
			for _, s := range tt.excludes {
				if strings.Contains(result, s) {
					t.Errorf("extractSummary() should not contain %q, got:\n%s", s, result)
				}
			}
		})
	}
}

func TestNewRunner(t *testing.T) {
	cfg := &config.Config{
		OutputDir: "./outputs",
	}

	runner := NewRunner(cfg, true)

	if runner.config != cfg {
		t.Error("Runner should store config reference")
	}
	if !runner.verbose {
		t.Error("Runner should store verbose flag")
	}
}

func TestRunSkipsInCreateMode(t *testing.T) {
	cfg := &config.Config{
		Mode:      config.ModeCreate,
		OutputDir: "./outputs",
		PostProcessing: config.PostProcessingConfig{
			GenerateDiffSummary:   true,
			GeneratePRDescription: true,
		},
	}

	runner := NewRunner(cfg, false)
	results := runner.Run()

	// Should skip all post-processing in create mode
	if len(results) != 0 {
		t.Errorf("Expected 0 results in create mode, got %d", len(results))
	}
}

func TestResultFields(t *testing.T) {
	result := Result{
		Step:    "test step",
		Success: true,
		Output:  "test output",
		Error:   nil,
	}

	if result.Step != "test step" {
		t.Errorf("Step = %q, want %q", result.Step, "test step")
	}
	if !result.Success {
		t.Error("Success should be true")
	}
	if result.Output != "test output" {
		t.Errorf("Output = %q, want %q", result.Output, "test output")
	}
	if result.Error != nil {
		t.Errorf("Error should be nil, got %v", result.Error)
	}
}

func TestExtractSummaryTrimsWhitespace(t *testing.T) {
	content := `# Title

   Indented content with spaces

## Next`

	result := extractSummary(content)

	// Should be trimmed
	if strings.HasPrefix(result, " ") || strings.HasSuffix(result, " ") {
		t.Errorf("Result should be trimmed, got: %q", result)
	}
}

func TestExtractSummaryPreservesInternalNewlines(t *testing.T) {
	content := `# Title

First paragraph.

Second paragraph.

## Next`

	result := extractSummary(content)

	// Should preserve paragraphs
	if !strings.Contains(result, "First paragraph.") {
		t.Error("Should contain first paragraph")
	}
	if !strings.Contains(result, "Second paragraph.") {
		t.Error("Should contain second paragraph")
	}
}
