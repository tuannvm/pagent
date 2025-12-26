package tui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DiscoverInputFolders returns folders that might contain input files
// Includes both top-level folders and their immediate subfolders
func DiscoverInputFolders() []string {
	candidates := []string{"inputs", "input", "examples", "specs"}
	var folders []string

	for _, dir := range candidates {
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			continue
		}

		// Add the top-level folder
		folders = append(folders, dir)

		// Also add immediate subfolders
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
				folders = append(folders, filepath.Join(dir, entry.Name()))
			}
		}
	}
	return folders
}

// DiscoverInputFiles finds potential input files in the current directory
// Returns files sorted by modification time (most recent first), limited to 10
func DiscoverInputFiles() []string {
	patterns := []string{
		"*.md",
		"*.yaml",
		"*.yml",
		"prd*",
		"PRD*",
		"requirements*",
	}

	fileSet := make(map[string]os.FileInfo)

	// Search current directory
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		for _, m := range matches {
			if info, err := os.Stat(m); err == nil && !info.IsDir() {
				fileSet[m] = info
			}
		}
	}

	// Also check common subdirectories (inputs first, NO docs - that's for documentation)
	subdirs := []string{"inputs", "input", "examples", "specs"}
	for _, dir := range subdirs {
		// Check if subdir exists
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}
		for _, pattern := range patterns {
			matches, _ := filepath.Glob(filepath.Join(dir, pattern))
			for _, m := range matches {
				if info, err := os.Stat(m); err == nil && !info.IsDir() {
					fileSet[m] = info
				}
			}
		}
	}

	// Convert to slice and sort by modification time (recent first)
	type fileWithTime struct {
		path    string
		modTime int64
	}
	var files []fileWithTime
	for path, info := range fileSet {
		files = append(files, fileWithTime{path, info.ModTime().Unix()})
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime > files[j].modTime
	})

	// Return paths only, limit to 10
	result := make([]string, 0, 10)
	for i, f := range files {
		if i >= 10 {
			break
		}
		result = append(result, f.path)
	}
	return result
}

// IsMarkdownOrYAML checks if file has a supported extension
func IsMarkdownOrYAML(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".md" || ext == ".yaml" || ext == ".yml"
}

// FileExists checks if a file exists and is not a directory
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// DirExists checks if a directory exists
func DirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
