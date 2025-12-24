package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewManager(t *testing.T) {
	m := NewManager("/output")

	if m.outputDir != "/output" {
		t.Errorf("outputDir = %q, want %q", m.outputDir, "/output")
	}
	if m.statePath != filepath.Join("/output", StateFile) {
		t.Errorf("statePath = %q, want %q", m.statePath, filepath.Join("/output", StateFile))
	}
	if m.state == nil || m.state.AgentOutputs == nil {
		t.Error("state should be initialized")
	}
}

func TestManagerLoadNoStateFile(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)

	// No state file exists - should succeed with empty state
	if err := m.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if m.state.InputHash != "" {
		t.Error("Fresh state should have empty InputHash")
	}
	if len(m.state.AgentOutputs) != 0 {
		t.Error("Fresh state should have no AgentOutputs")
	}
}

func TestManagerSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)

	// Set some state
	m.state.InputHash = "abc123"
	m.state.ConfigHash = "config456"
	m.state.AgentOutputs["test"] = AgentOutput{
		OutputPath: "/path/to/output.md",
		OutputHash: "out789",
	}

	// Save
	if err := m.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file exists
	statePath := filepath.Join(tmpDir, StateFile)
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Fatal("State file not created")
	}

	// Load into new manager
	m2 := NewManager(tmpDir)
	if err := m2.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if m2.state.InputHash != "abc123" {
		t.Errorf("InputHash = %q, want %q", m2.state.InputHash, "abc123")
	}
	if m2.state.ConfigHash != "config456" {
		t.Errorf("ConfigHash = %q, want %q", m2.state.ConfigHash, "config456")
	}
	if out, ok := m2.state.AgentOutputs["test"]; !ok || out.OutputHash != "out789" {
		t.Error("AgentOutputs not loaded correctly")
	}
}

func TestManagerUpdateInputHash(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)

	// Create test files
	file1 := filepath.Join(tmpDir, "a.md")
	file2 := filepath.Join(tmpDir, "b.md")
	os.WriteFile(file1, []byte("content a"), 0644)
	os.WriteFile(file2, []byte("content b"), 0644)

	if err := m.UpdateInputHash([]string{file1, file2}); err != nil {
		t.Fatalf("UpdateInputHash() error = %v", err)
	}

	if m.state.InputHash == "" {
		t.Error("InputHash should be set")
	}

	// Hash should be deterministic
	hash1 := m.state.InputHash
	m.UpdateInputHash([]string{file1, file2})
	if m.state.InputHash != hash1 {
		t.Error("Same inputs should produce same hash")
	}

	// Different order should produce same hash (sorted internally)
	m.UpdateInputHash([]string{file2, file1})
	if m.state.InputHash != hash1 {
		t.Error("Order should not affect hash")
	}
}

func TestManagerUpdateInputHashDetectsChanges(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)

	file := filepath.Join(tmpDir, "input.md")
	os.WriteFile(file, []byte("original"), 0644)

	m.UpdateInputHash([]string{file})
	hash1 := m.state.InputHash

	// Modify file
	os.WriteFile(file, []byte("modified"), 0644)

	m.UpdateInputHash([]string{file})
	hash2 := m.state.InputHash

	if hash1 == hash2 {
		t.Error("Different content should produce different hash")
	}
}

func TestManagerUpdateConfigHash(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)

	stack := map[string]string{"cloud": "aws"}
	prefs := map[string]bool{"stateless": true}

	if err := m.UpdateConfigHash("balanced", stack, prefs); err != nil {
		t.Fatalf("UpdateConfigHash() error = %v", err)
	}

	if m.state.ConfigHash == "" {
		t.Error("ConfigHash should be set")
	}

	// Same config should produce same hash
	hash1 := m.state.ConfigHash
	m.UpdateConfigHash("balanced", stack, prefs)
	if m.state.ConfigHash != hash1 {
		t.Error("Same config should produce same hash")
	}

	// Different persona should produce different hash
	m.UpdateConfigHash("production", stack, prefs)
	if m.state.ConfigHash == hash1 {
		t.Error("Different persona should produce different hash")
	}
}

func TestManagerRecordAgentOutput(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)

	// Set input/config hashes first
	inputFile := filepath.Join(tmpDir, "input.md")
	outputFile := filepath.Join(tmpDir, "output.md")
	os.WriteFile(inputFile, []byte("input"), 0644)
	os.WriteFile(outputFile, []byte("output"), 0644)

	m.UpdateInputHash([]string{inputFile})
	m.UpdateConfigHash("balanced", nil, nil)

	// Record output
	if err := m.RecordAgentOutput("architect", outputFile, []string{}); err != nil {
		t.Fatalf("RecordAgentOutput() error = %v", err)
	}

	out, ok := m.state.AgentOutputs["architect"]
	if !ok {
		t.Fatal("Agent output not recorded")
	}

	if out.OutputPath != outputFile {
		t.Errorf("OutputPath = %q, want %q", out.OutputPath, outputFile)
	}
	if out.OutputHash == "" {
		t.Error("OutputHash should be set")
	}
	if out.InputHashAtGeneration != m.state.InputHash {
		t.Error("InputHashAtGeneration should match current InputHash")
	}
	if out.ConfigHashAtGeneration != m.state.ConfigHash {
		t.Error("ConfigHashAtGeneration should match current ConfigHash")
	}
}

func TestManagerRecordAgentOutputWithDependencies(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)

	// Create outputs
	archOutput := filepath.Join(tmpDir, "architecture.md")
	implOutput := filepath.Join(tmpDir, "code/.complete")
	os.WriteFile(archOutput, []byte("arch"), 0644)
	os.MkdirAll(filepath.Join(tmpDir, "code"), 0755)
	os.WriteFile(implOutput, []byte("impl"), 0644)

	// Record architect first
	m.RecordAgentOutput("architect", archOutput, []string{})

	// Record implementer with architect as dependency
	if err := m.RecordAgentOutput("implementer", implOutput, []string{"architect"}); err != nil {
		t.Fatalf("RecordAgentOutput() error = %v", err)
	}

	implOut := m.state.AgentOutputs["implementer"]
	archHash := m.state.AgentOutputs["architect"].OutputHash

	if implOut.DependencyHashes["architect"] != archHash {
		t.Errorf("DependencyHashes[architect] = %q, want %q",
			implOut.DependencyHashes["architect"], archHash)
	}
}

func TestShouldRegenerateNoRecord(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)

	should, reason := m.ShouldRegenerate("test", "/path/output.md", nil)

	if !should {
		t.Error("Should regenerate when no previous record exists")
	}
	if reason != "no previous output recorded" {
		t.Errorf("Unexpected reason: %s", reason)
	}
}

func TestShouldRegenerateOutputMissing(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)

	// Record output but don't create the file
	m.state.AgentOutputs["test"] = AgentOutput{
		OutputPath: filepath.Join(tmpDir, "missing.md"),
		OutputHash: "abc",
	}

	should, reason := m.ShouldRegenerate("test", filepath.Join(tmpDir, "missing.md"), nil)

	if !should {
		t.Error("Should regenerate when output file is missing")
	}
	if reason != "output file does not exist" {
		t.Errorf("Unexpected reason: %s", reason)
	}
}

func TestShouldRegenerateOutputModified(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)

	outputFile := filepath.Join(tmpDir, "output.md")
	os.WriteFile(outputFile, []byte("original"), 0644)

	// Record with original hash
	m.RecordAgentOutput("test", outputFile, nil)

	// Modify the file (simulating user edit)
	os.WriteFile(outputFile, []byte("user modified"), 0644)

	should, reason := m.ShouldRegenerate("test", outputFile, nil)

	if !should {
		t.Error("Should regenerate when output was modified externally")
	}
	if reason != "output file was modified externally" {
		t.Errorf("Unexpected reason: %s", reason)
	}
}

func TestShouldRegenerateInputsChanged(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)

	inputFile := filepath.Join(tmpDir, "input.md")
	outputFile := filepath.Join(tmpDir, "output.md")
	os.WriteFile(inputFile, []byte("input"), 0644)
	os.WriteFile(outputFile, []byte("output"), 0644)

	// Set initial state
	m.UpdateInputHash([]string{inputFile})
	m.UpdateConfigHash("balanced", nil, nil)
	m.RecordAgentOutput("test", outputFile, nil)

	// Change input
	os.WriteFile(inputFile, []byte("modified input"), 0644)
	m.UpdateInputHash([]string{inputFile})

	should, reason := m.ShouldRegenerate("test", outputFile, nil)

	if !should {
		t.Error("Should regenerate when inputs changed")
	}
	if reason != "input files changed" {
		t.Errorf("Unexpected reason: %s", reason)
	}
}

func TestShouldRegenerateConfigChanged(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)

	inputFile := filepath.Join(tmpDir, "input.md")
	outputFile := filepath.Join(tmpDir, "output.md")
	os.WriteFile(inputFile, []byte("input"), 0644)
	os.WriteFile(outputFile, []byte("output"), 0644)

	// Set initial state
	m.UpdateInputHash([]string{inputFile})
	m.UpdateConfigHash("balanced", nil, nil)
	m.RecordAgentOutput("test", outputFile, nil)

	// Change config
	m.UpdateConfigHash("production", nil, nil)

	should, reason := m.ShouldRegenerate("test", outputFile, nil)

	if !should {
		t.Error("Should regenerate when config changed")
	}
	if reason != "configuration changed" {
		t.Errorf("Unexpected reason: %s", reason)
	}
}

func TestShouldRegenerateDependencyChanged(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)

	archOutput := filepath.Join(tmpDir, "architecture.md")
	implOutput := filepath.Join(tmpDir, "impl.md")
	os.WriteFile(archOutput, []byte("arch v1"), 0644)
	os.WriteFile(implOutput, []byte("impl"), 0644)

	// Set initial state
	m.UpdateInputHash(nil)
	m.UpdateConfigHash("balanced", nil, nil)
	m.RecordAgentOutput("architect", archOutput, nil)
	m.RecordAgentOutput("implementer", implOutput, []string{"architect"})

	// Change architect output
	os.WriteFile(archOutput, []byte("arch v2"), 0644)
	m.RecordAgentOutput("architect", archOutput, nil) // Re-record with new hash

	should, reason := m.ShouldRegenerate("implementer", implOutput, []string{"architect"})

	if !should {
		t.Error("Should regenerate when dependency changed")
	}
	if reason != "dependency architect output changed" {
		t.Errorf("Unexpected reason: %s", reason)
	}
}

func TestShouldRegenerateUpToDate(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)

	inputFile := filepath.Join(tmpDir, "input.md")
	outputFile := filepath.Join(tmpDir, "output.md")
	os.WriteFile(inputFile, []byte("input"), 0644)
	os.WriteFile(outputFile, []byte("output"), 0644)

	// Set state
	m.UpdateInputHash([]string{inputFile})
	m.UpdateConfigHash("balanced", nil, nil)
	m.RecordAgentOutput("test", outputFile, nil)

	// Nothing changed
	should, reason := m.ShouldRegenerate("test", outputFile, nil)

	if should {
		t.Errorf("Should NOT regenerate when up-to-date, reason: %s", reason)
	}
	if reason != "up-to-date" {
		t.Errorf("Expected 'up-to-date' reason, got: %s", reason)
	}
}

func TestShouldRegenerateMissingDependency(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)

	outputFile := filepath.Join(tmpDir, "output.md")
	os.WriteFile(outputFile, []byte("output"), 0644)

	m.UpdateInputHash(nil)
	m.UpdateConfigHash("balanced", nil, nil)
	// Record output but NOT the dependency
	m.RecordAgentOutput("implementer", outputFile, nil)

	// Check with dependency that wasn't recorded
	should, reason := m.ShouldRegenerate("implementer", outputFile, []string{"architect"})

	if !should {
		t.Error("Should regenerate when dependency has no recorded output")
	}
	if reason != "dependency architect has no recorded output" {
		t.Errorf("Unexpected reason: %s", reason)
	}
}

func TestClear(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)

	// Set some state and save
	m.state.InputHash = "test"
	m.state.AgentOutputs["test"] = AgentOutput{}
	m.Save()

	// Clear
	if err := m.Clear(); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	// State should be reset
	if m.state.InputHash != "" {
		t.Error("InputHash should be empty after Clear")
	}
	if len(m.state.AgentOutputs) != 0 {
		t.Error("AgentOutputs should be empty after Clear")
	}
}

func TestHashBytesIsDeterministic(t *testing.T) {
	data := []byte("test data")

	hash1 := hashBytes(data)
	hash2 := hashBytes(data)

	if hash1 != hash2 {
		t.Error("hashBytes should be deterministic")
	}
	if hash1 == "" {
		t.Error("hashBytes should return non-empty string")
	}
}

func TestHashFilesIncludesPath(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two files with same content but different paths
	file1 := filepath.Join(tmpDir, "a.md")
	file2 := filepath.Join(tmpDir, "b.md")
	os.WriteFile(file1, []byte("same content"), 0644)
	os.WriteFile(file2, []byte("same content"), 0644)

	hash1, _ := hashFiles([]string{file1})
	hash2, _ := hashFiles([]string{file2})

	// Hashes should differ because paths are included
	if hash1 == hash2 {
		t.Error("Different paths should produce different hashes even with same content")
	}
}
