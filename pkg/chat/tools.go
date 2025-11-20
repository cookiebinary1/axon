package chat

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/axon/pkg/fsctx"
	"github.com/axon/pkg/logger"
)

// resolvePath safely resolves a path relative to project root and validates it's within the root
func (s *Session) resolvePath(path string) (string, error) {
	return fsctx.ResolvePath(s.projectRoot, path)
}

// confirmAction asks the user to confirm an action interactively
func (s *Session) confirmAction(action, description string) (bool, error) {
	fmt.Printf("\n%s⚠️  WRITE OPERATION REQUESTED%s\n", colorYellow+colorBold, colorReset)
	fmt.Printf("%sAction:%s %s\n", colorCyan, colorReset, action)
	fmt.Printf("%sDetails:%s %s\n", colorCyan, colorReset, description)
	fmt.Printf("%sDo you want to proceed? [Y,n]:%s ", colorYellow+colorBold, colorReset)

	// Use the session's scanner for input
	if !s.scanner.Scan() {
		return false, fmt.Errorf("failed to read confirmation input")
	}

	response := strings.TrimSpace(strings.ToLower(s.scanner.Text()))
	// Empty input (just Enter) defaults to "yes"
	if response == "" {
		return true, nil
	}
	return response == "yes" || response == "y", nil
}

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
	case "write_file":
		result, err = s.toolWriteFile(args)
	case "create_file":
		result, err = s.toolCreateFile(args)
	case "update_file":
		result, err = s.toolUpdateFile(args)
	case "string_replace":
		result, err = s.toolStringReplace(args)
	case "create_directory":
		result, err = s.toolCreateDirectory(args)
	case "get_tree_list":
		result, err = s.toolGetTreeList(args)
	case "get_file_symbols":
		result, err = s.toolGetFileSymbols(args)
	case "delete_file":
		result, err = s.toolDeleteFile(args)
	case "delete_directory":
		result, err = s.toolDeleteDirectory(args)
	case "move_file":
		result, err = s.toolMoveFile(args)
	case "copy_file":
		result, err = s.toolCopyFile(args)
	case "find_files":
		result, err = s.toolFindFiles(args)
	case "find_files_by_extension":
		result, err = s.toolFindFilesByExtension(args)
	case "search_symbols":
		result, err = s.toolSearchSymbols(args)
	case "get_project_stats":
		result, err = s.toolGetProjectStats(args)
	case "get_file_info":
		result, err = s.toolGetFileInfo(args)
	case "find_dependencies":
		result, err = s.toolFindDependencies(args)
	case "git_status":
		result, err = s.toolGitStatus(args)
	case "git_diff":
		result, err = s.toolGitDiff(args)
	case "find_symbol_references":
		result, err = s.toolFindSymbolReferences(args)
	case "execute":
		result, err = s.toolExecute(args)
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

	fullPath, err := s.resolvePath(path)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

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
		resolved, err := s.resolvePath(searchPath)
		if err != nil {
			return "", fmt.Errorf("invalid path: %w", err)
		}
		searchPath = resolved
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

// toolWriteFile writes content to a file (creates new or overwrites existing)
func (s *Session) toolWriteFile(args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("path argument is required")
	}

	content, ok := args["content"].(string)
	if !ok {
		return "", fmt.Errorf("content argument is required")
	}

	// Check if file exists
	fullPath, err := s.resolvePath(path)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	fileExists := false
	if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
		fileExists = true
	}

	action := "Create new file"
	description := fmt.Sprintf("File: %s\nSize: ~%d bytes", path, len(content))
	if fileExists {
		action = "Overwrite existing file"
		description = fmt.Sprintf("File: %s (WILL BE OVERWRITTEN)\nNew size: ~%d bytes", path, len(content))
	}

	// Require confirmation
	confirmed, err := s.confirmAction(action, description)
	if err != nil {
		return "", fmt.Errorf("failed to get confirmation: %w", err)
	}
	if !confirmed {
		return `{"cancelled": true, "message": "User cancelled the operation"}`, nil
	}

	err = fsctx.WriteFile(s.projectRoot, path, content, s.cfg)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	result := map[string]interface{}{
		"path":    path,
		"success": true,
		"message": "File written successfully",
	}

	jsonResult, _ := json.Marshal(result)
	return string(jsonResult), nil
}

// toolCreateFile creates a new file (fails if file already exists)
func (s *Session) toolCreateFile(args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("path argument is required")
	}

	content, ok := args["content"].(string)
	if !ok {
		return "", fmt.Errorf("content argument is required")
	}

	fullPath, err := s.resolvePath(path)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	if _, err := os.Stat(fullPath); err == nil {
		return "", fmt.Errorf("file already exists: %s", path)
	}

	action := "Create new file"
	description := fmt.Sprintf("File: %s\nSize: ~%d bytes", path, len(content))

	// Require confirmation
	confirmed, err := s.confirmAction(action, description)
	if err != nil {
		return "", fmt.Errorf("failed to get confirmation: %w", err)
	}
	if !confirmed {
		return `{"cancelled": true, "message": "User cancelled the operation"}`, nil
	}

	err = fsctx.WriteFile(s.projectRoot, path, content, s.cfg)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}

	result := map[string]interface{}{
		"path":    path,
		"success": true,
		"message": "File created successfully",
	}

	jsonResult, _ := json.Marshal(result)
	return string(jsonResult), nil
}

// toolUpdateFile updates a file by replacing its entire content
func (s *Session) toolUpdateFile(args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("path argument is required")
	}

	content, ok := args["content"].(string)
	if !ok {
		return "", fmt.Errorf("content argument is required")
	}

	fullPath, err := s.resolvePath(path)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return "", fmt.Errorf("file does not exist: %s", path)
	}

	action := "Update file"
	description := fmt.Sprintf("File: %s\nNew size: ~%d bytes", path, len(content))

	// Require confirmation
	confirmed, err := s.confirmAction(action, description)
	if err != nil {
		return "", fmt.Errorf("failed to get confirmation: %w", err)
	}
	if !confirmed {
		return `{"cancelled": true, "message": "User cancelled the operation"}`, nil
	}

	err = fsctx.WriteFile(s.projectRoot, path, content, s.cfg)
	if err != nil {
		return "", fmt.Errorf("failed to update file: %w", err)
	}

	result := map[string]interface{}{
		"path":    path,
		"success": true,
		"message": "File updated successfully",
	}

	jsonResult, _ := json.Marshal(result)
	return string(jsonResult), nil
}

// toolStringReplace replaces a string pattern in a file
func (s *Session) toolStringReplace(args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("path argument is required")
	}

	oldStr, ok := args["old_string"].(string)
	if !ok {
		return "", fmt.Errorf("old_string argument is required")
	}

	newStr, ok := args["new_string"].(string)
	if !ok {
		return "", fmt.Errorf("new_string argument is required")
	}

	// Read file first
	content, _, err := fsctx.ReadFile(s.projectRoot, path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Check if old string exists
	if !strings.Contains(content, oldStr) {
		return "", fmt.Errorf("pattern not found in file: %s", path)
	}

	// Count occurrences
	count := strings.Count(content, oldStr)

	action := "Replace string in file"
	description := fmt.Sprintf("File: %s\nPattern occurrences: %d\nReplacing: %q\nWith: %q", path, count, oldStr, newStr)

	// Require confirmation
	confirmed, err := s.confirmAction(action, description)
	if err != nil {
		return "", fmt.Errorf("failed to get confirmation: %w", err)
	}
	if !confirmed {
		return `{"cancelled": true, "message": "User cancelled the operation"}`, nil
	}

	// Replace
	newContent := strings.ReplaceAll(content, oldStr, newStr)

	err = fsctx.WriteFile(s.projectRoot, path, newContent, s.cfg)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	result := map[string]interface{}{
		"path":         path,
		"success":      true,
		"replacements": count,
		"message":      fmt.Sprintf("Replaced %d occurrence(s)", count),
	}

	jsonResult, _ := json.Marshal(result)
	return string(jsonResult), nil
}

// toolCreateDirectory creates a directory
func (s *Session) toolCreateDirectory(args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("path argument is required")
	}

	fullPath, err := s.resolvePath(path)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	// Check if already exists
	if info, err := os.Stat(fullPath); err == nil {
		if info.IsDir() {
			return "", fmt.Errorf("directory already exists: %s", path)
		}
		return "", fmt.Errorf("path exists but is not a directory: %s", path)
	}

	action := "Create directory"
	description := fmt.Sprintf("Path: %s", path)

	// Require confirmation
	confirmed, err := s.confirmAction(action, description)
	if err != nil {
		return "", fmt.Errorf("failed to get confirmation: %w", err)
	}
	if !confirmed {
		return `{"cancelled": true, "message": "User cancelled the operation"}`, nil
	}

	err = os.MkdirAll(fullPath, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	result := map[string]interface{}{
		"path":    path,
		"success": true,
		"message": "Directory created successfully",
	}

	jsonResult, _ := json.Marshal(result)
	return string(jsonResult), nil
}

// toolGetTreeList returns a tree view of files and directories
func (s *Session) toolGetTreeList(args map[string]interface{}) (string, error) {
	path := ""
	if p, ok := args["path"].(string); ok {
		path = p
	}

	if s.index == nil {
		return "", fmt.Errorf("project index not available")
	}

	tree, err := s.index.GetTree(path)
	if err != nil {
		return "", fmt.Errorf("failed to get tree: %w", err)
	}

	result := map[string]interface{}{
		"path": path,
		"tree": tree,
	}

	jsonResult, _ := json.Marshal(result)
	return string(jsonResult), nil
}

// toolGetFileSymbols returns symbols (classes, functions) from a file
func (s *Session) toolGetFileSymbols(args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("path argument is required")
	}

	if s.index == nil {
		return "", fmt.Errorf("project index not available")
	}

	symbols, err := s.index.GetFileSymbols(path)
	if err != nil {
		return "", fmt.Errorf("failed to get file symbols: %w", err)
	}

	// Group symbols by type
	classes := []map[string]interface{}{}
	functions := []map[string]interface{}{}
	structs := []map[string]interface{}{}
	others := []map[string]interface{}{}

	for _, sym := range symbols {
		symData := map[string]interface{}{
			"name": sym.Name,
			"type": sym.Type,
			"line": sym.Line,
		}
		if sym.Signature != "" {
			symData["signature"] = sym.Signature
		}

		switch sym.Type {
		case "class", "interface":
			classes = append(classes, symData)
		case "struct":
			structs = append(structs, symData)
		case "function", "method":
			functions = append(functions, symData)
		default:
			others = append(others, symData)
		}
	}

	result := map[string]interface{}{
		"path":      path,
		"classes":   classes,
		"structs":   structs,
		"functions": functions,
		"others":    others,
		"total":     len(symbols),
	}

	jsonResult, _ := json.Marshal(result)
	return string(jsonResult), nil
}
