package fsctx

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/axon/pkg/project"
)

const (
	// MaxFileSize is the maximum file size to read fully (200 KB)
	MaxFileSize = 200 * 1024
)

// ReadFile reads a file relative to the project root, with size checking.
// If the file is too large, it returns a truncated version.
// Returns the content, a boolean indicating if it was truncated, and an error.
func ReadFile(projectRoot, filePath string) (content string, truncated bool, err error) {
	fullPath, err := ResolvePath(projectRoot, filePath)
	if err != nil {
		return "", false, fmt.Errorf("invalid path: %w", err)
	}

	// Check if file exists
	info, err := os.Stat(fullPath)
	if err != nil {
		return "", false, fmt.Errorf("file not found: %w", err)
	}

	if info.IsDir() {
		return "", false, fmt.Errorf("path is a directory, not a file")
	}

	// Check file size
	if info.Size() > MaxFileSize {
		// Read only first MaxFileSize bytes
		file, err := os.Open(fullPath)
		if err != nil {
			return "", false, fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()

		buf := make([]byte, MaxFileSize)
		n, err := file.Read(buf)
		if err != nil && err.Error() != "EOF" {
			return "", false, fmt.Errorf("failed to read file: %w", err)
		}

		return string(buf[:n]), true, nil
	}

	// Read full file
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", false, fmt.Errorf("failed to read file: %w", err)
	}

	return string(data), false, nil
}

// ReadFileRange reads a specific line range from a file.
// Line numbers are 1-based.
func ReadFileRange(projectRoot, filePath string, startLine, endLine int) (content string, err error) {
	fullPath, err := ResolvePath(projectRoot, filePath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	if startLine < 1 {
		startLine = 1
	}
	if endLine > len(lines) {
		endLine = len(lines)
	}
	if startLine > endLine {
		return "", fmt.Errorf("invalid range: start line %d is after end line %d", startLine, endLine)
	}

	selectedLines := lines[startLine-1 : endLine]
	return strings.Join(selectedLines, "\n"), nil
}

// ValidatePath ensures that a resolved path is within the project root.
// Returns an error if the path is outside the project root.
func ValidatePath(projectRoot, filePath string) error {
	projectRootAbs, err := filepath.Abs(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to resolve project root: %w", err)
	}

	filePathAbs, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to resolve file path: %w", err)
	}

	// Check that the file path is within project root
	relPath, err := filepath.Rel(projectRootAbs, filePathAbs)
	if err != nil {
		return fmt.Errorf("failed to compute relative path: %w", err)
	}

	// If relative path starts with "..", it's outside the project root
	if strings.HasPrefix(relPath, "..") {
		return fmt.Errorf("path is outside project root: %s", filePath)
	}

	return nil
}

// ResolvePath resolves a path relative to the project root.
// It validates that absolute paths are within the project root.
func ResolvePath(projectRoot, path string) (string, error) {
	var fullPath string
	if filepath.IsAbs(path) {
		// For absolute paths, validate they're within project root
		fullPath = path
		if err := ValidatePath(projectRoot, fullPath); err != nil {
			return "", err
		}
	} else {
		fullPath = filepath.Join(projectRoot, path)
	}

	// Always validate the final path
	if err := ValidatePath(projectRoot, fullPath); err != nil {
		return "", err
	}

	return fullPath, nil
}

// ShouldIgnore checks if a path should be ignored based on the config's ignore patterns.
func ShouldIgnore(path string, cfg *project.Config) bool {
	// Normalize path separators
	normalizedPath := strings.ReplaceAll(path, "\\", "/")
	return project.ShouldIgnore(normalizedPath, cfg.Context.Ignore)
}

// GetFileExtension returns the file extension (e.g., ".go", ".php")
// Returns empty string for hidden files starting with dot that have no extension part
func GetFileExtension(filePath string) string {
	base := filepath.Base(filePath)
	if strings.HasPrefix(base, ".") && !strings.Contains(base[1:], ".") {
		// Hidden file with no extension (e.g., .gitignore)
		return ""
	}
	ext := filepath.Ext(filePath)
	return strings.ToLower(ext)
}

// WriteFile writes content to a file relative to the project root.
// It validates that the path is safe (not ignored, within project root).
// Creates parent directories if they don't exist.
func WriteFile(projectRoot, filePath string, content string, cfg *project.Config) error {
	// Normalize path separators
	normalizedPath := strings.ReplaceAll(filePath, "\\", "/")

	// Check if path should be ignored
	if ShouldIgnore(normalizedPath, cfg) {
		return fmt.Errorf("cannot write to ignored path: %s", filePath)
	}

	fullPath, err := ResolvePath(projectRoot, filePath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Create parent directories if they don't exist
	parentDir := filepath.Dir(fullPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directories: %w", err)
	}

	// Write file
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
