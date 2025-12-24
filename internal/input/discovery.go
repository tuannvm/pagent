package input

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SupportedExtensions lists file extensions to include as input
var SupportedExtensions = []string{
	".md",   // Markdown (PRDs, requirements, docs)
	".yaml", // YAML specs
	".yml",  // YAML specs
	".json", // JSON specs (OpenAPI, etc.)
	".txt",  // Plain text requirements
}

// Input represents discovered input files
type Input struct {
	// IsDirectory indicates if input was a directory
	IsDirectory bool
	// Path is the original input path (file or directory)
	Path string
	// Files contains all discovered input files (absolute paths)
	Files []string
	// PrimaryFile is the main input file (first .md file or first file)
	PrimaryFile string
}

// Discover scans the input path and returns discovered input files
// If path is a file, returns that single file
// If path is a directory, scans for supported file types
func Discover(path string) (*Input, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("path not found: %w", err)
	}

	if !info.IsDir() {
		// Single file input
		return &Input{
			IsDirectory: false,
			Path:        absPath,
			Files:       []string{absPath},
			PrimaryFile: absPath,
		}, nil
	}

	// Directory input - scan for files
	files, err := scanDirectory(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to scan directory: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no supported input files found in %s (supported: %v)", absPath, SupportedExtensions)
	}

	// Determine primary file (first .md file, or first file overall)
	primary := findPrimaryFile(files)

	return &Input{
		IsDirectory: true,
		Path:        absPath,
		Files:       files,
		PrimaryFile: primary,
	}, nil
}

// scanDirectory recursively scans a directory for supported files
func scanDirectory(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Skip hidden directories
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}

		// Skip directories and hidden files
		if info.IsDir() || strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		// Check if file has supported extension
		ext := strings.ToLower(filepath.Ext(path))
		for _, supported := range SupportedExtensions {
			if ext == supported {
				files = append(files, path)
				break
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort files for consistent ordering
	sort.Strings(files)

	return files, nil
}

// findPrimaryFile determines the primary input file
// Priority: files with "prd" in name > .md files > first file
func findPrimaryFile(files []string) string {
	// First, look for files with "prd" in the name
	for _, f := range files {
		name := strings.ToLower(filepath.Base(f))
		if strings.Contains(name, "prd") && strings.HasSuffix(name, ".md") {
			return f
		}
	}

	// Then, look for any .md file
	for _, f := range files {
		if strings.HasSuffix(strings.ToLower(f), ".md") {
			return f
		}
	}

	// Fall back to first file
	if len(files) > 0 {
		return files[0]
	}

	return ""
}

// RelativePaths returns file paths relative to the input directory
func (i *Input) RelativePaths() []string {
	if !i.IsDirectory {
		return []string{filepath.Base(i.PrimaryFile)}
	}

	var rel []string
	for _, f := range i.Files {
		r, err := filepath.Rel(i.Path, f)
		if err != nil {
			r = filepath.Base(f)
		}
		rel = append(rel, r)
	}
	return rel
}

// Summary returns a human-readable summary of discovered inputs
func (i *Input) Summary() string {
	if !i.IsDirectory {
		return fmt.Sprintf("Input: %s", filepath.Base(i.PrimaryFile))
	}

	byExt := make(map[string]int)
	for _, f := range i.Files {
		ext := filepath.Ext(f)
		byExt[ext]++
	}

	var parts []string
	for ext, count := range byExt {
		parts = append(parts, fmt.Sprintf("%d %s", count, ext))
	}
	sort.Strings(parts)

	return fmt.Sprintf("Input: %s (%d files: %s)", filepath.Base(i.Path), len(i.Files), strings.Join(parts, ", "))
}
