package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindProjectRoot(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()

	// Create a nested directory
	nestedDir := filepath.Join(tmpDir, "subdir", "nested")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("Failed to create nested directory: %v", err)
	}

	// Test 1: Start from nested dir, should find .git at root
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git directory: %v", err)
	}

	root, err := FindProjectRoot(nestedDir)
	if err != nil {
		t.Fatalf("FindProjectRoot failed: %v", err)
	}
	if root != tmpDir {
		t.Errorf("Expected project root %s, got %s", tmpDir, root)
	}

	// Test 2: Find .axon.yml
	tmpDir2 := t.TempDir()
	nestedDir2 := filepath.Join(tmpDir2, "subdir", "nested")
	if err := os.MkdirAll(nestedDir2, 0755); err != nil {
		t.Fatalf("Failed to create nested directory: %v", err)
	}

	axonYmlPath := filepath.Join(tmpDir2, ".axon.yml")
	if err := os.WriteFile(axonYmlPath, []byte("llm:\n  base_url: http://test\n"), 0644); err != nil {
		t.Fatalf("Failed to create .axon.yml: %v", err)
	}

	root2, err := FindProjectRoot(nestedDir2)
	if err != nil {
		t.Fatalf("FindProjectRoot failed: %v", err)
	}
	if root2 != tmpDir2 {
		t.Errorf("Expected project root %s, got %s", tmpDir2, root2)
	}

	// Test 3: No .git or .axon.yml, should return startDir
	tmpDir3 := t.TempDir()
	nestedDir3 := filepath.Join(tmpDir3, "subdir", "nested")
	if err := os.MkdirAll(nestedDir3, 0755); err != nil {
		t.Fatalf("Failed to create nested directory: %v", err)
	}

	root3, err := FindProjectRoot(nestedDir3)
	if err != nil {
		t.Fatalf("FindProjectRoot failed: %v", err)
	}
	absNestedDir3, _ := filepath.Abs(nestedDir3)
	if root3 != absNestedDir3 {
		t.Errorf("Expected project root %s (abs of start dir), got %s", absNestedDir3, root3)
	}
}

func TestLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Test with config file
	configYaml := `llm:
  base_url: "http://test:9090"
  model: "test-model"
  temperature: 0.5
context:
  ignore:
    - "test/"
    - "temp/"
`
	axonYmlPath := filepath.Join(tmpDir, ".axon.yml")
	if err := os.WriteFile(axonYmlPath, []byte(configYaml), 0644); err != nil {
		t.Fatalf("Failed to create .axon.yml: %v", err)
	}

	cfg, err := LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.LLM.BaseURL != "http://test:9090" {
		t.Errorf("Expected base URL http://test:9090, got %s", cfg.LLM.BaseURL)
	}
	if cfg.LLM.Model != "test-model" {
		t.Errorf("Expected model test-model, got %s", cfg.LLM.Model)
	}
	if cfg.LLM.Temperature != 0.5 {
		t.Errorf("Expected temperature 0.5, got %f", cfg.LLM.Temperature)
	}
	if len(cfg.Context.Ignore) != 2 {
		t.Errorf("Expected 2 ignore patterns, got %d", len(cfg.Context.Ignore))
	}

	// Test defaults when no config file
	tmpDir2 := t.TempDir()
	cfg2, err := LoadConfig(tmpDir2)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg2.LLM.BaseURL != "http://127.0.0.1:8080" {
		t.Errorf("Expected default base URL http://127.0.0.1:8080, got %s", cfg2.LLM.BaseURL)
	}
	if cfg2.LLM.Model != "qwen2.5-coder-3b" {
		t.Errorf("Expected default model qwen2.5-coder-3b, got %s", cfg2.LLM.Model)
	}
}

func TestShouldIgnore(t *testing.T) {
	tests := []struct {
		path     string
		patterns []string
		expected bool
	}{
		{"vendor/file.go", []string{"vendor/"}, true},
		{"src/file.go", []string{"vendor/"}, false},
		{"node_modules/test", []string{"node_modules/"}, true},
		{"app/file.go", []string{"vendor/", "node_modules/"}, false},
		{"storage/logs/app.log", []string{"storage/"}, true},
		{".git/config", []string{".git/"}, true},
	}

	for _, tt := range tests {
		result := ShouldIgnore(tt.path, tt.patterns)
		if result != tt.expected {
			t.Errorf("ShouldIgnore(%q, %v) = %v, expected %v", tt.path, tt.patterns, result, tt.expected)
		}
	}
}
