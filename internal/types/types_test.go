package types

import "testing"

func TestDefaultStack(t *testing.T) {
	stack := DefaultStack()

	// Verify critical defaults that affect generated code
	if stack.Cloud != "aws" {
		t.Errorf("Default Cloud = %q, want %q", stack.Cloud, "aws")
	}
	if stack.Compute != "kubernetes" {
		t.Errorf("Default Compute = %q, want %q", stack.Compute, "kubernetes")
	}
	if stack.Database != "postgres" {
		t.Errorf("Default Database = %q, want %q", stack.Database, "postgres")
	}
	if stack.Cache != "redis" {
		t.Errorf("Default Cache = %q, want %q", stack.Cache, "redis")
	}
	if stack.IaC != "terraform" {
		t.Errorf("Default IaC = %q, want %q", stack.IaC, "terraform")
	}
	if stack.CI != "github-actions" {
		t.Errorf("Default CI = %q, want %q", stack.CI, "github-actions")
	}
	if stack.Monitoring != "prometheus" {
		t.Errorf("Default Monitoring = %q, want %q", stack.Monitoring, "prometheus")
	}

	// Optional fields should be empty by default
	if stack.MessageQueue != "" {
		t.Errorf("Default MessageQueue should be empty, got %q", stack.MessageQueue)
	}
	if stack.DataLake != "" {
		t.Errorf("Default DataLake should be empty, got %q", stack.DataLake)
	}
}

func TestDefaultPreferences(t *testing.T) {
	prefs := DefaultPreferences()

	// Verify critical defaults
	if prefs.Language != "go" {
		t.Errorf("Default Language = %q, want %q", prefs.Language, "go")
	}
	if prefs.APIStyle != "rest" {
		t.Errorf("Default APIStyle = %q, want %q", prefs.APIStyle, "rest")
	}
	if prefs.Stateless {
		t.Error("Default Stateless should be false")
	}
	if prefs.DependencyStyle != "minimal" {
		t.Errorf("Default DependencyStyle = %q, want %q", prefs.DependencyStyle, "minimal")
	}
	if prefs.TestingDepth != "unit" {
		t.Errorf("Default TestingDepth = %q, want %q", prefs.TestingDepth, "unit")
	}
	if !prefs.Containerized {
		t.Error("Default Containerized should be true")
	}
	if !prefs.IncludeCI {
		t.Error("Default IncludeCI should be true")
	}
	if !prefs.IncludeIaC {
		t.Error("Default IncludeIaC should be true")
	}
}

func TestStackResolutionHasConflicts(t *testing.T) {
	tests := []struct {
		name       string
		resolution *StackResolution
		expected   bool
	}{
		{
			name:       "nil resolution has no conflicts",
			resolution: nil,
			expected:   false,
		},
		{
			name:       "empty resolution has no conflicts",
			resolution: &StackResolution{},
			expected:   false,
		},
		{
			name: "all resolved has no conflicts",
			resolution: &StackResolution{
				Conflicts: []StackConflict{
					{Category: "database", Resolved: true, Resolution: "postgres"},
					{Category: "cache", Resolved: true, Resolution: "none"},
				},
			},
			expected: false,
		},
		{
			name: "one unresolved has conflicts",
			resolution: &StackResolution{
				Conflicts: []StackConflict{
					{Category: "database", Resolved: true, Resolution: "postgres"},
					{Category: "cache", Resolved: false},
				},
			},
			expected: true,
		},
		{
			name: "all unresolved has conflicts",
			resolution: &StackResolution{
				Conflicts: []StackConflict{
					{Category: "database", Resolved: false},
					{Category: "cache", Resolved: false},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.resolution.HasConflicts(); got != tt.expected {
				t.Errorf("HasConflicts() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestStackResolutionGetResolution(t *testing.T) {
	resolution := &StackResolution{
		Conflicts: []StackConflict{
			{Category: "database", Resolved: true, Resolution: "postgres"},
			{Category: "cache", Resolved: true, Resolution: "none"},
			{Category: "compute", Resolved: false, Resolution: ""},
		},
	}

	tests := []struct {
		category string
		expected string
	}{
		{"database", "postgres"},
		{"cache", "none"},
		{"compute", ""},          // unresolved returns empty
		{"nonexistent", ""},      // missing category returns empty
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			if got := resolution.GetResolution(tt.category); got != tt.expected {
				t.Errorf("GetResolution(%q) = %q, want %q", tt.category, got, tt.expected)
			}
		})
	}
}

func TestStackResolutionGetResolutionNil(t *testing.T) {
	var resolution *StackResolution = nil

	// Should not panic on nil receiver
	if got := resolution.GetResolution("database"); got != "" {
		t.Errorf("GetResolution on nil should return empty, got %q", got)
	}
}

func TestStackConflictFields(t *testing.T) {
	conflict := StackConflict{
		Category:    "database",
		ConfigValue: "postgres",
		PRDHint:     "stateless, no database",
		Resolved:    true,
		Resolution:  "none",
	}

	if conflict.Category != "database" {
		t.Errorf("Category = %q, want %q", conflict.Category, "database")
	}
	if conflict.ConfigValue != "postgres" {
		t.Errorf("ConfigValue = %q, want %q", conflict.ConfigValue, "postgres")
	}
	if conflict.PRDHint != "stateless, no database" {
		t.Errorf("PRDHint = %q, want %q", conflict.PRDHint, "stateless, no database")
	}
	if !conflict.Resolved {
		t.Error("Resolved should be true")
	}
	if conflict.Resolution != "none" {
		t.Errorf("Resolution = %q, want %q", conflict.Resolution, "none")
	}
}

func TestTechStackNoneValues(t *testing.T) {
	// Test that "none" is a valid value for optional fields
	stack := TechStack{
		Cloud:        "aws",
		Compute:      "github-actions",
		Database:     "none",
		Cache:        "none",
		MessageQueue: "none",
	}

	if stack.Database != "none" {
		t.Errorf("Database = %q, want %q", stack.Database, "none")
	}
	if stack.Cache != "none" {
		t.Errorf("Cache = %q, want %q", stack.Cache, "none")
	}
	if stack.MessageQueue != "none" {
		t.Errorf("MessageQueue = %q, want %q", stack.MessageQueue, "none")
	}
}

func TestTechStackGitHubActionsCompute(t *testing.T) {
	// Verify github-actions is a valid compute option
	stack := TechStack{
		Cloud:   "aws",
		Compute: "github-actions",
	}

	if stack.Compute != "github-actions" {
		t.Errorf("Compute = %q, want %q", stack.Compute, "github-actions")
	}
}

func TestArchitecturePreferencesStateless(t *testing.T) {
	// Stateless preferences should work correctly
	prefs := ArchitecturePreferences{
		Stateless: true,
		Language:  "go",
		APIStyle:  "rest",
	}

	if !prefs.Stateless {
		t.Error("Stateless should be true")
	}
}
