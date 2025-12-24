package input

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverSingleFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "prd.md")
	if err := os.WriteFile(filePath, []byte("# PRD"), 0644); err != nil {
		t.Fatal(err)
	}

	input, err := Discover(filePath)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if input.IsDirectory {
		t.Error("Expected IsDirectory = false for single file")
	}
	if len(input.Files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(input.Files))
	}
	if input.PrimaryFile != input.Files[0] {
		t.Error("PrimaryFile should equal the single file")
	}
}

func TestDiscoverDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create various files
	files := map[string]string{
		"requirements.md":  "# Requirements",
		"api-spec.yaml":    "openapi: 3.0.0",
		"data.json":        "{}",
		"notes.txt":        "notes",
		"ignored.go":       "package main", // unsupported extension
		".hidden.md":       "hidden",       // hidden file
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	input, err := Discover(tmpDir)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if !input.IsDirectory {
		t.Error("Expected IsDirectory = true")
	}

	// Should find 4 supported files (not .go, not hidden)
	if len(input.Files) != 4 {
		t.Errorf("Expected 4 files, got %d: %v", len(input.Files), input.Files)
	}

	// Should not include .go files
	for _, f := range input.Files {
		if strings.HasSuffix(f, ".go") {
			t.Errorf("Should not include .go file: %s", f)
		}
		if strings.HasPrefix(filepath.Base(f), ".") {
			t.Errorf("Should not include hidden file: %s", f)
		}
	}
}

func TestDiscoverEmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := Discover(tmpDir)
	if err == nil {
		t.Error("Discover() should return error for empty directory")
	}
	if !strings.Contains(err.Error(), "no supported input files") {
		t.Errorf("Expected 'no supported input files' error, got: %v", err)
	}
}

func TestDiscoverNonexistentPath(t *testing.T) {
	_, err := Discover("/nonexistent/path")
	if err == nil {
		t.Error("Discover() should return error for nonexistent path")
	}
}

func TestDiscoverSkipsHiddenDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a hidden directory with files
	hiddenDir := filepath.Join(tmpDir, ".hidden")
	if err := os.MkdirAll(hiddenDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hiddenDir, "secret.md"), []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a visible file
	if err := os.WriteFile(filepath.Join(tmpDir, "visible.md"), []byte("visible"), 0644); err != nil {
		t.Fatal(err)
	}

	input, err := Discover(tmpDir)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(input.Files) != 1 {
		t.Errorf("Expected 1 file (hidden dir should be skipped), got %d", len(input.Files))
	}
}

func TestFindPrimaryFilePRDPriority(t *testing.T) {
	// PRD files should be prioritized
	files := []string{
		"/path/requirements.md",
		"/path/my-prd.md",
		"/path/api-spec.yaml",
	}

	primary := findPrimaryFile(files)

	if !strings.Contains(primary, "prd") {
		t.Errorf("Expected PRD file to be primary, got %s", primary)
	}
}

func TestFindPrimaryFileMDPriority(t *testing.T) {
	// .md files should be prioritized over others when no PRD
	files := []string{
		"/path/api-spec.yaml",
		"/path/requirements.md",
		"/path/data.json",
	}

	primary := findPrimaryFile(files)

	if !strings.HasSuffix(primary, ".md") {
		t.Errorf("Expected .md file to be primary, got %s", primary)
	}
}

func TestFindPrimaryFileFallback(t *testing.T) {
	// First file when no .md files
	files := []string{
		"/path/api-spec.yaml",
		"/path/data.json",
	}

	primary := findPrimaryFile(files)

	if primary != "/path/api-spec.yaml" {
		t.Errorf("Expected first file as fallback, got %s", primary)
	}
}

func TestFindPrimaryFileEmpty(t *testing.T) {
	primary := findPrimaryFile([]string{})
	if primary != "" {
		t.Errorf("Expected empty string for empty input, got %s", primary)
	}
}

func TestRelativePathsSingleFile(t *testing.T) {
	input := &Input{
		IsDirectory: false,
		PrimaryFile: "/path/to/prd.md",
		Files:       []string{"/path/to/prd.md"},
	}

	rel := input.RelativePaths()

	if len(rel) != 1 || rel[0] != "prd.md" {
		t.Errorf("RelativePaths() = %v, want [prd.md]", rel)
	}
}

func TestRelativePathsDirectory(t *testing.T) {
	input := &Input{
		IsDirectory: true,
		Path:        "/base",
		Files:       []string{"/base/a.md", "/base/sub/b.md"},
	}

	rel := input.RelativePaths()

	if len(rel) != 2 {
		t.Fatalf("Expected 2 relative paths, got %d", len(rel))
	}
	if rel[0] != "a.md" {
		t.Errorf("RelativePaths()[0] = %q, want %q", rel[0], "a.md")
	}
	if rel[1] != filepath.Join("sub", "b.md") {
		t.Errorf("RelativePaths()[1] = %q, want %q", rel[1], filepath.Join("sub", "b.md"))
	}
}

func TestSummarySingleFile(t *testing.T) {
	input := &Input{
		IsDirectory: false,
		PrimaryFile: "/path/to/my-prd.md",
	}

	summary := input.Summary()

	if !strings.Contains(summary, "my-prd.md") {
		t.Errorf("Summary should contain filename, got: %s", summary)
	}
}

func TestSummaryDirectory(t *testing.T) {
	input := &Input{
		IsDirectory: true,
		Path:        "/path/to/specs",
		Files: []string{
			"/path/to/specs/a.md",
			"/path/to/specs/b.md",
			"/path/to/specs/c.yaml",
		},
	}

	summary := input.Summary()

	if !strings.Contains(summary, "3 files") {
		t.Errorf("Summary should mention file count, got: %s", summary)
	}
	if !strings.Contains(summary, ".md") {
		t.Errorf("Summary should mention extensions, got: %s", summary)
	}
}

func TestDiscoverNestedDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested structure
	nestedDir := filepath.Join(tmpDir, "sub", "nested")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create files at different levels
	os.WriteFile(filepath.Join(tmpDir, "root.md"), []byte("root"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "sub", "sub.md"), []byte("sub"), 0644)
	os.WriteFile(filepath.Join(nestedDir, "nested.md"), []byte("nested"), 0644)

	input, err := Discover(tmpDir)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	// Should find all 3 files
	if len(input.Files) != 3 {
		t.Errorf("Expected 3 files from nested dirs, got %d", len(input.Files))
	}
}

func TestDiscoverFilesAreSorted(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files in non-alphabetical order
	files := []string{"zebra.md", "alpha.md", "middle.md"}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, f), []byte(f), 0644); err != nil {
			t.Fatal(err)
		}
	}

	input, err := Discover(tmpDir)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	// Files should be sorted
	for i := 1; i < len(input.Files); i++ {
		if input.Files[i] < input.Files[i-1] {
			t.Errorf("Files not sorted: %v", input.Files)
			break
		}
	}
}

func TestSupportedExtensions(t *testing.T) {
	// Verify expected extensions are supported
	expected := []string{".md", ".yaml", ".yml", ".json", ".txt"}

	for _, ext := range expected {
		found := false
		for _, supported := range SupportedExtensions {
			if ext == supported {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected %s to be a supported extension", ext)
		}
	}
}

func TestDiscoverCaseInsensitiveExtensions(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files with uppercase extensions
	os.WriteFile(filepath.Join(tmpDir, "file.MD"), []byte("MD"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file.YAML"), []byte("YAML"), 0644)

	input, err := Discover(tmpDir)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	// Should find both files (case insensitive matching)
	if len(input.Files) != 2 {
		t.Errorf("Expected 2 files with uppercase extensions, got %d", len(input.Files))
	}
}
