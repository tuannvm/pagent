package postprocess

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tuannvm/pagent/internal/config"
)

// Runner handles post-processing tasks after agents complete
type Runner struct {
	config  *config.Config
	verbose bool
}

// NewRunner creates a new post-processing runner
func NewRunner(cfg *config.Config, verbose bool) *Runner {
	return &Runner{
		config:  cfg,
		verbose: verbose,
	}
}

// Result holds the result of a post-processing step
type Result struct {
	Step    string
	Success bool
	Output  string
	Error   error
}

// Run executes all configured post-processing steps
func (r *Runner) Run() []Result {
	var results []Result

	// Only run post-processing in modify mode
	if !r.config.IsModifyMode() {
		return results
	}

	pp := r.config.PostProcessing

	// Run validation commands first
	if len(pp.ValidationCommands) > 0 {
		for _, cmd := range pp.ValidationCommands {
			result := r.runValidationCommand(cmd)
			results = append(results, result)
			// Stop on first validation failure
			if !result.Success {
				return results
			}
		}
	}

	// Generate diff summary
	if pp.GenerateDiffSummary {
		result := r.generateDiffSummary()
		results = append(results, result)
	}

	// Generate PR description
	if pp.GeneratePRDescription {
		result := r.generatePRDescription()
		results = append(results, result)
	}

	return results
}

// runValidationCommand executes a validation command in the target codebase
func (r *Runner) runValidationCommand(cmdStr string) Result {
	result := Result{
		Step: fmt.Sprintf("validate: %s", cmdStr),
	}

	// Run command in target codebase directory
	workDir := r.config.TargetCodebase
	if workDir == "" {
		workDir = "."
	}

	// Parse command string
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		result.Error = fmt.Errorf("empty command")
		return result
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Dir = workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if r.verbose {
		fmt.Printf("[POST] Running: %s (in %s)\n", cmdStr, workDir)
	}

	err := cmd.Run()
	result.Output = stdout.String()
	if stderr.Len() > 0 {
		result.Output += "\n" + stderr.String()
	}

	if err != nil {
		result.Error = fmt.Errorf("command failed: %w\n%s", err, result.Output)
		result.Success = false
	} else {
		result.Success = true
	}

	return result
}

// generateDiffSummary creates a git diff summary of changes
func (r *Runner) generateDiffSummary() Result {
	result := Result{
		Step: "generate diff summary",
	}

	workDir := r.config.TargetCodebase
	if workDir == "" {
		result.Error = fmt.Errorf("no target codebase specified")
		return result
	}

	// Get git diff stats
	cmd := exec.Command("git", "diff", "--stat", "HEAD")
	cmd.Dir = workDir

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		result.Error = fmt.Errorf("git diff failed: %w", err)
		return result
	}

	diffStat := stdout.String()
	if strings.TrimSpace(diffStat) == "" {
		result.Output = "No changes detected"
		result.Success = true
		return result
	}

	// Get detailed diff
	cmd = exec.Command("git", "diff", "HEAD")
	cmd.Dir = workDir
	stdout.Reset()
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		result.Error = fmt.Errorf("git diff failed: %w", err)
		return result
	}

	fullDiff := stdout.String()

	// Write diff summary to specs output dir
	specsDir := r.config.GetEffectiveSpecsOutputDir()
	if err := os.MkdirAll(specsDir, 0755); err != nil {
		result.Error = fmt.Errorf("failed to create specs dir: %w", err)
		return result
	}

	summaryPath := filepath.Join(specsDir, "diff-summary.md")
	content := fmt.Sprintf(`# Diff Summary

## Changes Overview
%s

## Full Diff
%s
`, "```\n"+diffStat+"```", "```diff\n"+fullDiff+"```")

	if err := os.WriteFile(summaryPath, []byte(content), 0644); err != nil {
		result.Error = fmt.Errorf("failed to write diff summary: %w", err)
		return result
	}

	result.Output = summaryPath
	result.Success = true
	return result
}

// generatePRDescription creates a PR description from the changes
func (r *Runner) generatePRDescription() Result {
	result := Result{
		Step: "generate PR description",
	}

	workDir := r.config.TargetCodebase
	if workDir == "" {
		result.Error = fmt.Errorf("no target codebase specified")
		return result
	}

	// Get list of changed files
	cmd := exec.Command("git", "diff", "--name-only", "HEAD")
	cmd.Dir = workDir

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		result.Error = fmt.Errorf("git diff failed: %w", err)
		return result
	}

	changedFiles := strings.TrimSpace(stdout.String())
	if changedFiles == "" {
		result.Output = "No changes to describe"
		result.Success = true
		return result
	}

	// Get diff stat for summary
	cmd = exec.Command("git", "diff", "--stat", "HEAD")
	cmd.Dir = workDir
	stdout.Reset()
	cmd.Stdout = &stdout

	_ = cmd.Run()
	diffStat := stdout.String()

	// Read architecture.md for context (if exists)
	specsDir := r.config.GetEffectiveSpecsOutputDir()
	archPath := filepath.Join(specsDir, "architecture.md")
	archContent := ""
	if data, err := os.ReadFile(archPath); err == nil {
		// Extract summary section if possible
		archContent = extractSummary(string(data))
	}

	// Generate PR description
	prDescPath := filepath.Join(specsDir, "pr-description.md")
	content := fmt.Sprintf(`# Pull Request Description

## Summary

<!-- Generated by pagent - please review and customize -->

This PR implements changes as described in the architecture specification.

%s

## Changes

### Files Modified
%s

### Diff Statistics
%s

## Testing

- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Manual testing completed

## Checklist

- [ ] Code follows project conventions
- [ ] Documentation updated (if needed)
- [ ] Security considerations addressed
`, archContent, "```\n"+changedFiles+"```", "```\n"+diffStat+"```")

	if err := os.WriteFile(prDescPath, []byte(content), 0644); err != nil {
		result.Error = fmt.Errorf("failed to write PR description: %w", err)
		return result
	}

	result.Output = prDescPath
	result.Success = true
	return result
}

// extractSummary extracts the first meaningful section from a markdown document
func extractSummary(content string) string {
	lines := strings.Split(content, "\n")
	var summary []string
	inSummary := false
	foundHeader := false

	for _, line := range lines {
		// Skip until we find the first heading
		if !foundHeader {
			if strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "## ") {
				foundHeader = true
				inSummary = true
			}
			continue
		}

		// Stop at the next major heading
		if inSummary && (strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "## ")) {
			break
		}

		if inSummary {
			summary = append(summary, line)
		}

		// Limit summary length
		if len(summary) > 20 {
			summary = append(summary, "...")
			break
		}
	}

	return strings.TrimSpace(strings.Join(summary, "\n"))
}
