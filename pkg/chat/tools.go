package chat

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/axon/pkg/fsctx"
	"github.com/axon/pkg/logger"
)

// ExecuteTool executes a tool call and returns the result
func (s *Session) ExecuteTool(name string, args map[string]interface{}) (string, error) {
	// Log tool execution
	argsJSON, _ := json.Marshal(args)
	logger.Logf("🔧 EXECUTING TOOL: %s with args: %s\n", name, string(argsJSON))

	var result string
	var err error

	switch name {
	case "read_file":
		result, err = s.toolReadFile(args)
	case "list_directory":
		result, err = s.toolListDirectory(args)
	case "grep":
		result, err = s.toolGrep(args)
	case "read_file_lines":
		result, err = s.toolReadFileLines(args)
	default:
		err = fmt.Errorf("unknown tool: %s", name)
	}

	// Log result
	logger.LogToolCall(name, string(argsJSON), result, err)

	return result, err
}

// toolReadFile reads a file
func (s *Session) toolReadFile(args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("path argument is required")
	}

	content, truncated, err := fsctx.ReadFile(s.projectRoot, path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	result := map[string]interface{}{
		"path":      path,
		"content":   content,
		"truncated": truncated,
	}

	if truncated {
		result["note"] = "File was truncated to first 200KB"
	}

	jsonResult, _ := json.Marshal(result)
	return string(jsonResult), nil
}

// toolListDirectory lists files in a directory
func (s *Session) toolListDirectory(args map[string]interface{}) (string, error) {
	path := ""
	if p, ok := args["path"].(string); ok {
		path = p
	}

	fullPath := filepath.Join(s.projectRoot, path)
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read directory: %w", err)
	}

	var files []map[string]interface{}
	var dirs []map[string]interface{}

	for _, entry := range entries {
		// Skip hidden files/dirs
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		item := map[string]interface{}{
			"name": entry.Name(),
			"size": info.Size(),
		}

		if entry.IsDir() {
			dirs = append(dirs, item)
		} else {
			item["extension"] = fsctx.GetFileExtension(entry.Name())
			files = append(files, item)
		}
	}

	result := map[string]interface{}{
		"path":  path,
		"files": files,
		"dirs":  dirs,
	}

	jsonResult, _ := json.Marshal(result)
	return string(jsonResult), nil
}

// toolGrep searches for a pattern in files
func (s *Session) toolGrep(args map[string]interface{}) (string, error) {
	pattern, ok := args["pattern"].(string)
	if !ok || pattern == "" {
		return "", fmt.Errorf("pattern argument is required")
	}

	searchPath := ""
	if p, ok := args["path"].(string); ok {
		searchPath = p
	}

	if searchPath == "" {
		searchPath = s.projectRoot
	} else {
		searchPath = filepath.Join(s.projectRoot, searchPath)
	}

	// Use ripgrep if available, otherwise grep
	var cmd *exec.Cmd

	if _, err := exec.LookPath("rg"); err == nil {
		cmd = exec.Command("rg", "-n", "--color", "never", pattern, searchPath)
	} else {
		cmd = exec.Command("grep", "-rn", "--color=never", pattern, searchPath)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		exitError, ok := err.(*exec.ExitError)
		if !ok || exitError.ExitCode() != 1 {
			return "", fmt.Errorf("search command failed: %w", err)
		}
		// Exit code 1 means no matches - that's OK
		output = []byte{}
	}

	matches := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(matches) == 1 && matches[0] == "" {
		matches = []string{}
	}

	result := map[string]interface{}{
		"pattern": pattern,
		"path":    searchPath,
		"matches": matches,
		"count":   len(matches),
	}

	jsonResult, _ := json.Marshal(result)
	return string(jsonResult), nil
}

// toolReadFileLines reads specific lines from a file
func (s *Session) toolReadFileLines(args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("path argument is required")
	}

	var startLine, endLine int

	if sl, ok := args["start_line"].(float64); ok {
		startLine = int(sl)
	} else if sl, ok := args["start_line"].(string); ok {
		var err error
		startLine, err = strconv.Atoi(sl)
		if err != nil {
			return "", fmt.Errorf("invalid start_line: %w", err)
		}
	} else {
		return "", fmt.Errorf("start_line argument is required")
	}

	if el, ok := args["end_line"].(float64); ok {
		endLine = int(el)
	} else if el, ok := args["end_line"].(string); ok {
		var err error
		endLine, err = strconv.Atoi(el)
		if err != nil {
			return "", fmt.Errorf("invalid end_line: %w", err)
		}
	} else {
		return "", fmt.Errorf("end_line argument is required")
	}

	content, err := fsctx.ReadFileRange(s.projectRoot, path, startLine, endLine)
	if err != nil {
		return "", fmt.Errorf("failed to read file lines: %w", err)
	}

	result := map[string]interface{}{
		"path":       path,
		"start_line": startLine,
		"end_line":   endLine,
		"content":    content,
	}

	jsonResult, _ := json.Marshal(result)
	return string(jsonResult), nil
}
