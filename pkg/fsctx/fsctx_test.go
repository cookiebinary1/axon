package fsctx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadFile(t *testing.T) {
	tmpDir := t.TempDir()
	projectRoot := tmpDir

	// Create a test file
	testFile := "test.txt"
	testContent := "Hello, World!"
	filePath := filepath.Join(projectRoot, testFile)
	if err := os.WriteFile(filePath, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test reading file
	content, truncated, err := ReadFile(projectRoot, testFile)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if content != testContent {
		t.Errorf("Expected content %q, got %q", testContent, content)
	}
	if truncated {
		t.Errorf("Expected not truncated, but it was")
	}

	// Test non-existent file
	_, _, err = ReadFile(projectRoot, "nonexistent.txt")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}

	// Test directory
	if err := os.MkdirAll(filepath.Join(projectRoot, "dir"), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	_, _, err = ReadFile(projectRoot, "dir")
	if err == nil {
		t.Error("Expected error for directory, got nil")
	}
}

func TestReadFileRange(t *testing.T) {
	tmpDir := t.TempDir()
	projectRoot := tmpDir

	// Create a test file with multiple lines
	testFile := "multiline.txt"
	testContent := "line1\nline2\nline3\nline4\nline5"
	filePath := filepath.Join(projectRoot, testFile)
	if err := os.WriteFile(filePath, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test reading range
	content, err := ReadFileRange(projectRoot, testFile, 2, 4)
	if err != nil {
		t.Fatalf("ReadFileRange failed: %v", err)
	}
	expected := "line2\nline3\nline4"
	if content != expected {
		t.Errorf("Expected content %q, got %q", expected, content)
	}

	// Test invalid range
	_, err = ReadFileRange(projectRoot, testFile, 5, 2)
	if err == nil {
		t.Error("Expected error for invalid range, got nil")
	}
}

func TestResolvePath(t *testing.T) {
	tmpDir := t.TempDir()
	projectRoot := tmpDir

	tests := []struct {
		path        string
		expected    string
		expectError bool
	}{
		{"relative/path.go", filepath.Join(projectRoot, "relative/path.go"), false},
		{"../outside.go", "", true},    // Should reject paths outside project root
		{"../../etc/passwd", "", true}, // Should reject path traversal
	}

	for _, tt := range tests {
		result, err := ResolvePath(projectRoot, tt.path)
		if tt.expectError {
			if err == nil {
				t.Errorf("ResolvePath(%q, %q) expected error, got nil", projectRoot, tt.path)
			}
		} else {
			if err != nil {
				t.Errorf("ResolvePath(%q, %q) unexpected error: %v", projectRoot, tt.path, err)
			} else if result != tt.expected {
				t.Errorf("ResolvePath(%q, %q) = %q, expected %q", projectRoot, tt.path, result, tt.expected)
			}
		}
	}
}

func TestGetFileExtension(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"file.go", ".go"},
		{"file.php", ".php"},
		{"file.js", ".js"},
		{"file.TS", ".ts"}, // Test case insensitivity
		{"file", ""},
		{".gitignore", ""},          // No extension after dot
		{"file.backup.txt", ".txt"}, // Last extension
	}

	for _, tt := range tests {
		result := GetFileExtension(tt.path)
		if result != tt.expected {
			t.Errorf("GetFileExtension(%q) = %q, expected %q", tt.path, result, tt.expected)
		}
	}
}
