package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsValidPersona(t *testing.T) {
	tests := []struct {
		name     string
		persona  string
		expected bool
	}{
		{"minimal is valid", "minimal", true},
		{"balanced is valid", "balanced", true},
		{"production is valid", "production", true},
		{"empty is invalid", "", false},
		{"unknown is invalid", "enterprise", false},
		{"case sensitive", "Balanced", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidPersona(tt.persona); got != tt.expected {
				t.Errorf("IsValidPersona(%q) = %v, want %v", tt.persona, got, tt.expected)
			}
		})
	}
}

func TestIsValidMode(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		expected bool
	}{
		{"create is valid", "create", true},
		{"modify is valid", "modify", true},
		{"empty defaults to create (valid)", "", true},
		{"unknown is invalid", "update", false},
		{"case sensitive", "Create", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidMode(tt.mode); got != tt.expected {
				t.Errorf("IsValidMode(%q) = %v, want %v", tt.mode, got, tt.expected)
			}
		})
	}
}

func TestConfigIsModifyMode(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		expected bool
	}{
		{"modify mode returns true", ModeModify, true},
		{"create mode returns false", ModeCreate, false},
		{"empty returns false", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Mode: tt.mode}
			if got := cfg.IsModifyMode(); got != tt.expected {
				t.Errorf("Config{Mode: %q}.IsModifyMode() = %v, want %v", tt.mode, got, tt.expected)
			}
		})
	}
}

func TestGetEffectiveCodeOutputDir(t *testing.T) {
	tests := []struct {
		name           string
		mode           string
		targetCodebase string
		outputDir      string
		expected       string
	}{
		{
			name:      "create mode uses output_dir/code",
			mode:      ModeCreate,
			outputDir: "/outputs",
			expected:  "/outputs/code",
		},
		{
			name:           "modify mode uses target_codebase",
			mode:           ModeModify,
			targetCodebase: "/existing/project",
			outputDir:      "/outputs",
			expected:       "/existing/project",
		},
		{
			name:      "modify mode without target falls back to output_dir/code",
			mode:      ModeModify,
			outputDir: "/outputs",
			expected:  "/outputs/code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Mode:           tt.mode,
				TargetCodebase: tt.targetCodebase,
				OutputDir:      tt.outputDir,
			}
			if got := cfg.GetEffectiveCodeOutputDir(); got != tt.expected {
				t.Errorf("GetEffectiveCodeOutputDir() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestGetEffectiveSpecsOutputDir(t *testing.T) {
	tests := []struct {
		name           string
		mode           string
		targetCodebase string
		specsOutputDir string
		outputDir      string
		expected       string
	}{
		{
			name:           "explicit specs_output_dir takes precedence",
			specsOutputDir: "/custom/specs",
			outputDir:      "/outputs",
			expected:       "/custom/specs",
		},
		{
			name:      "create mode uses output_dir",
			mode:      ModeCreate,
			outputDir: "/outputs",
			expected:  "/outputs",
		},
		{
			name:           "modify mode uses target/.pagent/specs",
			mode:           ModeModify,
			targetCodebase: "/existing/project",
			outputDir:      "/outputs",
			expected:       "/existing/project/.pagent/specs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Mode:           tt.mode,
				TargetCodebase: tt.targetCodebase,
				SpecsOutputDir: tt.specsOutputDir,
				OutputDir:      tt.outputDir,
			}
			if got := cfg.GetEffectiveSpecsOutputDir(); got != tt.expected {
				t.Errorf("GetEffectiveSpecsOutputDir() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDefault(t *testing.T) {
	cfg := Default()

	// Verify critical defaults
	if cfg.OutputDir != "./outputs" {
		t.Errorf("Default OutputDir = %q, want %q", cfg.OutputDir, "./outputs")
	}
	if cfg.Persona != PersonaBalanced {
		t.Errorf("Default Persona = %q, want %q", cfg.Persona, PersonaBalanced)
	}
	if cfg.Mode != ModeCreate {
		t.Errorf("Default Mode = %q, want %q", cfg.Mode, ModeCreate)
	}

	// Verify default agents exist
	expectedAgents := []string{"architect", "qa", "security", "implementer", "verifier"}
	for _, agent := range expectedAgents {
		if _, ok := cfg.Agents[agent]; !ok {
			t.Errorf("Default config missing agent %q", agent)
		}
	}

	// Verify dependency chain
	if len(cfg.Agents["architect"].DependsOn) != 0 {
		t.Error("architect should have no dependencies")
	}
	qaDepends := cfg.Agents["qa"].DependsOn
	if len(qaDepends) != 1 || qaDepends[0] != "architect" {
		t.Errorf("qa.DependsOn = %v, want [architect]", qaDepends)
	}
}

func TestGetAgentNames(t *testing.T) {
	cfg := &Config{
		Agents: map[string]AgentConfig{
			"zulu":    {},
			"alpha":   {},
			"charlie": {},
		},
	}

	names := cfg.GetAgentNames()

	// Should be sorted alphabetically
	expected := []string{"alpha", "charlie", "zulu"}
	if len(names) != len(expected) {
		t.Fatalf("GetAgentNames() returned %d names, want %d", len(names), len(expected))
	}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("GetAgentNames()[%d] = %q, want %q", i, name, expected[i])
		}
	}
}

func TestGetDependencies(t *testing.T) {
	cfg := &Config{
		Agents: map[string]AgentConfig{
			"first":  {DependsOn: []string{}},
			"second": {DependsOn: []string{"first"}},
			"third":  {DependsOn: []string{"first", "second"}},
		},
	}

	tests := []struct {
		agent    string
		expected []string
	}{
		{"first", []string{}},
		{"second", []string{"first"}},
		{"third", []string{"first", "second"}},
		{"nonexistent", nil},
	}

	for _, tt := range tests {
		t.Run(tt.agent, func(t *testing.T) {
			deps := cfg.GetDependencies(tt.agent)
			if tt.expected == nil {
				if deps != nil {
					t.Errorf("GetDependencies(%q) = %v, want nil", tt.agent, deps)
				}
				return
			}
			if len(deps) != len(tt.expected) {
				t.Errorf("GetDependencies(%q) = %v, want %v", tt.agent, deps, tt.expected)
			}
		})
	}
}

func TestLoadWithDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Minimal config - should get defaults applied
	minimalConfig := `
output_dir: ./test-outputs
agents:
  custom:
    output: custom.md
`
	if err := os.WriteFile(configPath, []byte(minimalConfig), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Check defaults were applied
	if cfg.Timeout != 300 {
		t.Errorf("Default timeout not applied: got %d, want 300", cfg.Timeout)
	}
	if cfg.Persona != PersonaBalanced {
		t.Errorf("Default persona not applied: got %q, want %q", cfg.Persona, PersonaBalanced)
	}
	if cfg.Mode != ModeCreate {
		t.Errorf("Default mode not applied: got %q, want %q", cfg.Mode, ModeCreate)
	}
	// Custom agents override defaults
	if _, ok := cfg.Agents["custom"]; !ok {
		t.Error("Custom agent not loaded")
	}
}

func TestLoadInvalidPersona(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	invalidConfig := `
output_dir: ./outputs
persona: enterprise
`
	if err := os.WriteFile(configPath, []byte(invalidConfig), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Load() should return error for invalid persona")
	}
}

func TestLoadInvalidMode(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	invalidConfig := `
output_dir: ./outputs
mode: update
`
	if err := os.WriteFile(configPath, []byte(invalidConfig), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Load() should return error for invalid mode")
	}
}

func TestLoadModifyModeRequiresTargetCodebase(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Modify mode without target_codebase
	invalidConfig := `
output_dir: ./outputs
mode: modify
`
	if err := os.WriteFile(configPath, []byte(invalidConfig), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Load() should return error when modify mode lacks target_codebase")
	}
}

func TestLoadModifyModeNonexistentTarget(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Modify mode with nonexistent target
	invalidConfig := `
output_dir: ./outputs
mode: modify
target_codebase: /nonexistent/path/that/does/not/exist
`
	if err := os.WriteFile(configPath, []byte(invalidConfig), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Load() should return error for nonexistent target_codebase")
	}
}

func TestLoadModifyModeValidTarget(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create a valid target directory
	targetDir := filepath.Join(tmpDir, "target")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}

	validConfig := `
output_dir: ./outputs
mode: modify
target_codebase: ` + targetDir + `
`
	if err := os.WriteFile(configPath, []byte(validConfig), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !cfg.IsModifyMode() {
		t.Error("Expected modify mode to be set")
	}
	if cfg.TargetCodebase != targetDir {
		t.Errorf("TargetCodebase = %q, want %q", cfg.TargetCodebase, targetDir)
	}
}

func TestApplyEnvOverrides(t *testing.T) {
	cfg := &Config{
		OutputDir: "./default",
		Timeout:   300,
	}

	// Set env vars using t.Setenv (auto cleanup)
	t.Setenv("PAGENT_OUTPUT_DIR", "/from/env")
	t.Setenv("PAGENT_TIMEOUT", "600")

	cfg.ApplyEnvOverrides()

	if cfg.OutputDir != "/from/env" {
		t.Errorf("OutputDir = %q, want %q", cfg.OutputDir, "/from/env")
	}
	if cfg.Timeout != 600 {
		t.Errorf("Timeout = %d, want 600", cfg.Timeout)
	}
}

func TestApplyEnvOverridesInvalidTimeout(t *testing.T) {
	cfg := &Config{
		OutputDir: "./default",
		Timeout:   300,
	}

	// Invalid timeout should be ignored
	t.Setenv("PAGENT_TIMEOUT", "invalid")

	cfg.ApplyEnvOverrides()

	if cfg.Timeout != 300 {
		t.Errorf("Invalid timeout should be ignored, got %d", cfg.Timeout)
	}
}

func TestLoadNonexistentFile(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Error("Load() should return error for nonexistent file")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Use truly invalid YAML with mismatched brackets/structure
	invalidYAML := `
output_dir: ./outputs
agents:
  - name: [unclosed bracket
  broken: {no closing brace
`
	if err := os.WriteFile(configPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Load() should return error for invalid YAML")
	}
}
