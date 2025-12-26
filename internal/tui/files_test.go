package tui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverInputFolders(t *testing.T) {
	// Create a temp directory with some test folders
	tmpDir, err := os.MkdirTemp("", "tui-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Save current directory and change to temp
	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	// Create test folders with subfolders
	os.Mkdir("inputs", 0755)
	os.Mkdir("examples", 0755)
	os.MkdirAll("examples/project-a", 0755)
	os.MkdirAll("examples/project-b", 0755)
	os.Mkdir("random", 0755) // Should not be discovered

	folders := DiscoverInputFolders()

	// Should find: inputs, examples, examples/project-a, examples/project-b
	if len(folders) != 4 {
		t.Errorf("expected 4 folders, got %d: %v", len(folders), folders)
	}

	// Check that random is not included
	for _, f := range folders {
		if f == "random" {
			t.Error("should not include 'random' folder")
		}
	}

	// Check subfolders are included
	hasSubfolder := false
	for _, f := range folders {
		if f == filepath.Join("examples", "project-a") {
			hasSubfolder = true
			break
		}
	}
	if !hasSubfolder {
		t.Error("should include subfolders like examples/project-a")
	}
}

func TestDiscoverInputFiles(t *testing.T) {
	// Create a temp directory with some test files
	tmpDir, err := os.MkdirTemp("", "tui-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Save current directory and change to temp
	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	// Create test files
	testFiles := []string{"prd.md", "requirements.yaml", "test.txt", "README.md"}
	for _, f := range testFiles {
		if err := os.WriteFile(f, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create test file %s: %v", f, err)
		}
	}

	// Create a subdirectory with files
	os.Mkdir("docs", 0755)
	os.WriteFile(filepath.Join("docs", "spec.md"), []byte("test"), 0644)

	// Test discovery
	files := DiscoverInputFiles()

	// Should find md and yaml files
	if len(files) == 0 {
		t.Error("expected to find some files, got none")
	}

	// Check that .txt file is not included (doesn't match patterns)
	for _, f := range files {
		if f == "test.txt" {
			t.Error("should not include .txt files that don't match patterns")
		}
	}
}

func TestIsMarkdownOrYAML(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"test.md", true},
		{"test.MD", true},
		{"test.yaml", true},
		{"test.YAML", true},
		{"test.yml", true},
		{"test.YML", true},
		{"test.go", false},
		{"test.txt", false},
		{"test.json", false},
		{"test", false},
		{".md", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := IsMarkdownOrYAML(tt.path)
			if result != tt.expected {
				t.Errorf("IsMarkdownOrYAML(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	// Create a temp file
	tmpFile, err := os.CreateTemp("", "tui-test-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "tui-test-dir-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"existing file", tmpFile.Name(), true},
		{"non-existent file", "/nonexistent/path/file.txt", false},
		{"directory", tmpDir, false}, // FileExists should return false for directories
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FileExists(tt.path)
			if result != tt.expected {
				t.Errorf("FileExists(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestDirExists(t *testing.T) {
	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "tui-test-dir-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a temp file
	tmpFile, err := os.CreateTemp("", "tui-test-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"existing directory", tmpDir, true},
		{"non-existent directory", "/nonexistent/path/dir", false},
		{"file", tmpFile.Name(), false}, // DirExists should return false for files
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DirExists(tt.path)
			if result != tt.expected {
				t.Errorf("DirExists(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}
