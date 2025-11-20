package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the axon configuration
type Config struct {
	LLM struct {
		BaseURL     string  `yaml:"base_url"`
		Model       string  `yaml:"model"`
		Temperature float64 `yaml:"temperature"`
	} `yaml:"llm"`
	Server struct {
		AutoStart  bool   `yaml:"auto_start"`
		ServerPath string `yaml:"server_path"`
		Model      string `yaml:"model"` // Model for llama-server
	} `yaml:"server"`
	Context struct {
		Ignore []string `yaml:"ignore"`
	} `yaml:"context"`
}

// FindProjectRoot walks upwards from startDir to find the project root.
// It looks for .git directory or .axon.yml/.axon.yaml file.
// If neither is found, returns startDir as the project root.
func FindProjectRoot(startDir string) (string, error) {
	absPath, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	dir := absPath
	for {
		// Check for .git directory
		gitPath := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitPath); err == nil && info.IsDir() {
			return dir, nil
		}

		// Check for .axon.yml
		axonYmlPath := filepath.Join(dir, ".axon.yml")
		if _, err := os.Stat(axonYmlPath); err == nil {
			return dir, nil
		}

		// Check for .axon.yaml
		axonYamlPath := filepath.Join(dir, ".axon.yaml")
		if _, err := os.Stat(axonYamlPath); err == nil {
			return dir, nil
		}

		// Move to parent directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root, stop
			break
		}
		dir = parent
	}

	// Neither .git nor .axon.yml found, return startDir as root
	return absPath, nil
}

// LoadConfig loads configuration from .axon.yml/.axon.yaml in the project root,
// then applies environment variable overrides.
func LoadConfig(projectRoot string) (*Config, error) {
	cfg := &Config{}

	// Set defaults
	cfg.LLM.BaseURL = "http://127.0.0.1:8080"
	cfg.LLM.Model = "qwen2.5-coder-3b"
	cfg.LLM.Temperature = 0.15
	cfg.Server.AutoStart = true                                     // Auto-start server by default
	cfg.Server.ServerPath = ""                                      // Use llama-server from PATH
	cfg.Server.Model = "Qwen/Qwen2.5-Coder-3B-Instruct-GGUF:Q4_K_M" // Default to 3B model

	// Try to load .axon.yml first
	axonYmlPath := filepath.Join(projectRoot, ".axon.yml")
	if data, err := os.ReadFile(axonYmlPath); err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse .axon.yml: %w", err)
		}
	} else {
		// Try .axon.yaml
		axonYamlPath := filepath.Join(projectRoot, ".axon.yaml")
		if data, err := os.ReadFile(axonYamlPath); err == nil {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				return nil, fmt.Errorf("failed to parse .axon.yaml: %w", err)
			}
		}
	}

	// Apply environment variable overrides
	if baseURL := os.Getenv("AXON_LLM_BASE_URL"); baseURL != "" {
		cfg.LLM.BaseURL = baseURL
	}
	if model := os.Getenv("AXON_LLM_MODEL"); model != "" {
		cfg.LLM.Model = model
	}
	if tempStr := os.Getenv("AXON_LLM_TEMPERATURE"); tempStr != "" {
		var temp float64
		if _, err := fmt.Sscanf(tempStr, "%f", &temp); err == nil {
			cfg.LLM.Temperature = temp
		}
	}
	// Server configuration
	if autoStart := os.Getenv("AXON_SERVER_AUTO_START"); autoStart != "" {
		cfg.Server.AutoStart = autoStart == "1" || autoStart == "true"
	}
	if serverPath := os.Getenv("AXON_SERVER_PATH"); serverPath != "" {
		cfg.Server.ServerPath = serverPath
	}
	if serverModel := os.Getenv("AXON_SERVER_MODEL"); serverModel != "" {
		cfg.Server.Model = serverModel
	}

	// Ensure ignore list has default values if empty
	if len(cfg.Context.Ignore) == 0 {
		cfg.Context.Ignore = []string{
			"vendor/",
			"node_modules/",
			"storage/",
			".git/",
		}
	}

	return cfg, nil
}

// ShouldIgnore checks if a path should be ignored based on the ignore patterns.
// Patterns can be simple strings (prefix match) or glob patterns.
func ShouldIgnore(path string, ignorePatterns []string) bool {
	for _, pattern := range ignorePatterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}

		// Simple prefix match
		if strings.HasPrefix(path, pattern) {
			return true
		}

		// Glob match
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err == nil && matched {
			return true
		}

		// Try matching against full path relative components
		if strings.Contains(path, pattern) {
			return true
		}
	}
	return false
}
