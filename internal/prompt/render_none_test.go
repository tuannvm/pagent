package prompt

import (
	"strings"
	"testing"

	"github.com/tuannvm/pagent/internal/types"
)

func TestRenderWithNoneValues(t *testing.T) {
	loader := NewLoader("")
	
	// Test with "none" values like oncall config
	vars := Variables{
		PRDPath:   "test-prd.md",
		OutputDir: "./outputs",
		OutputPath: "./outputs/architecture.md",
		Persona:   "balanced",
		Stack: types.TechStack{
			Cloud:        "aws",
			Compute:      "github-actions",
			Database:     "none",
			Cache:        "none",
			MessageQueue: "none",
			Monitoring:   "prometheus",
			Logging:      "loki",
			Chat:         "slack",
			CI:           "github-actions",
		},
		Preferences: types.ArchitecturePreferences{
			Language:      "go",
			APIStyle:      "rest",
			Stateless:     false, // Will be inferred from none values
			IncludeCI:     true,
			IncludeIaC:    true,
			Containerized: true,
		},
	}
	
	// Test helper methods
	t.Run("IsStateless inferred from none values", func(t *testing.T) {
		if !vars.IsStateless() {
			t.Error("IsStateless should be true when all storage is none")
		}
	})
	
	t.Run("HasDatabase false for none", func(t *testing.T) {
		if vars.HasDatabase() {
			t.Error("HasDatabase should be false when database is none")
		}
	})
	
	t.Run("HasCache false for none", func(t *testing.T) {
		if vars.HasCache() {
			t.Error("HasCache should be false when cache is none")
		}
	})
	
	t.Run("HasMessageQueue false for none", func(t *testing.T) {
		if vars.HasMessageQueue() {
			t.Error("HasMessageQueue should be false when message_queue is none")
		}
	})
	
	t.Run("IsGitHubActions true", func(t *testing.T) {
		if !vars.IsGitHubActions() {
			t.Error("IsGitHubActions should be true")
		}
	})
	
	t.Run("IsKubernetes false", func(t *testing.T) {
		if vars.IsKubernetes() {
			t.Error("IsKubernetes should be false for github-actions")
		}
	})
	
	t.Run("NeedsContainerization false for github-actions", func(t *testing.T) {
		if vars.NeedsContainerization() {
			t.Error("NeedsContainerization should be false for github-actions")
		}
	})
	
	t.Run("architect.md renders without errors", func(t *testing.T) {
		rendered, err := loader.LoadAndRender("architect", "", "", vars)
		if err != nil {
			t.Fatalf("Error rendering architect: %v", err)
		}
		
		if strings.Contains(rendered, "{{") {
			t.Error("Unrendered template syntax found in architect.md")
		}
		
		// Should show "None" with warning for database
		if !strings.Contains(rendered, "⚠️ Stateless - no database") {
			t.Error("Expected stateless warning for database")
		}
	})
	
	t.Run("implementer.md renders without errors", func(t *testing.T) {
		rendered, err := loader.LoadAndRender("implementer", "", "", vars)
		if err != nil {
			t.Fatalf("Error rendering implementer: %v", err)
		}
		
		if strings.Contains(rendered, "{{") {
			t.Error("Unrendered template syntax found in implementer.md")
		}
		
		// Should NOT mention pgx when database is none
		if strings.Contains(rendered, "Use pgx for PostgreSQL") {
			t.Error("pgx should not be mentioned when database is none")
		}
	})
}

func TestRenderWithRealDatabase(t *testing.T) {
	loader := NewLoader("")
	
	vars := Variables{
		PRDPath:   "test-prd.md",
		OutputDir: "./outputs",
		OutputPath: "./outputs/architecture.md",
		Persona:   "balanced",
		Stack: types.TechStack{
			Cloud:        "aws",
			Compute:      "kubernetes",
			Database:     "postgres",
			Cache:        "redis",
			MessageQueue: "kafka",
			Monitoring:   "prometheus",
		},
		Preferences: types.ArchitecturePreferences{
			Language:      "go",
			Stateless:     false,
			IncludeCI:     true,
			IncludeIaC:    true,
			Containerized: true,
		},
	}
	
	t.Run("HasDatabase true for postgres", func(t *testing.T) {
		if !vars.HasDatabase() {
			t.Error("HasDatabase should be true for postgres")
		}
	})
	
	t.Run("IsStateless false with database", func(t *testing.T) {
		if vars.IsStateless() {
			t.Error("IsStateless should be false when database is configured")
		}
	})
	
	t.Run("implementer.md mentions pgx for postgres", func(t *testing.T) {
		rendered, err := loader.LoadAndRender("implementer", "", "", vars)
		if err != nil {
			t.Fatalf("Error rendering: %v", err)
		}
		
		if !strings.Contains(rendered, "pgx") {
			t.Error("pgx should be mentioned when database is postgres")
		}
	})
}
