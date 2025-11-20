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
)

// toolDeleteFile deletes a file
func (s *Session) toolDeleteFile(args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("path argument is required")
	}

	fullPath, err := s.resolvePath(path)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return "", fmt.Errorf("file does not exist: %s", path)
	}

	action := "Delete file"
	description := fmt.Sprintf("File: %s\n⚠️  This action cannot be undone!", path)

	confirmed, err := s.confirmAction(action, description)
	if err != nil {
		return "", fmt.Errorf("failed to get confirmation: %w", err)
	}
	if !confirmed {
		return `{"cancelled": true, "message": "User cancelled the operation"}`, nil
	}

	err = os.Remove(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to delete file: %w", err)
	}

	result := map[string]interface{}{
		"path":    path,
		"success": true,
		"message": "File deleted successfully",
	}

	jsonResult, _ := json.Marshal(result)
	return string(jsonResult), nil
}

// toolDeleteDirectory deletes a directory
func (s *Session) toolDeleteDirectory(args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("path argument is required")
	}

	fullPath, err := s.resolvePath(path)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	info, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("directory does not exist: %s", path)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", path)
	}

	action := "Delete directory"
	description := fmt.Sprintf("Directory: %s\n⚠️  This will delete the directory and ALL its contents! This action cannot be undone!", path)

	confirmed, err := s.confirmAction(action, description)
	if err != nil {
		return "", fmt.Errorf("failed to get confirmation: %w", err)
	}
	if !confirmed {
		return `{"cancelled": true, "message": "User cancelled the operation"}`, nil
	}

	err = os.RemoveAll(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to delete directory: %w", err)
	}

	result := map[string]interface{}{
		"path":    path,
		"success": true,
		"message": "Directory deleted successfully",
	}

	jsonResult, _ := json.Marshal(result)
	return string(jsonResult), nil
}

// toolMoveFile moves or renames a file
func (s *Session) toolMoveFile(args map[string]interface{}) (string, error) {
	source, ok := args["source"].(string)
	if !ok || source == "" {
		return "", fmt.Errorf("source argument is required")
	}

	destination, ok := args["destination"].(string)
	if !ok || destination == "" {
		return "", fmt.Errorf("destination argument is required")
	}

	sourcePath, err := s.resolvePath(source)
	if err != nil {
		return "", fmt.Errorf("invalid source path: %w", err)
	}

	destPath, err := s.resolvePath(destination)
	if err != nil {
		return "", fmt.Errorf("invalid destination path: %w", err)
	}

	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return "", fmt.Errorf("source file does not exist: %s", source)
	}

	destExists := false
	if _, err := os.Stat(destPath); err == nil {
		destExists = true
	}

	action := "Move/rename file"
	description := fmt.Sprintf("Source: %s\nDestination: %s", source, destination)
	if destExists {
		description += "\n⚠️  Destination file exists and will be overwritten!"
	}

	confirmed, err := s.confirmAction(action, description)
	if err != nil {
		return "", fmt.Errorf("failed to get confirmation: %w", err)
	}
	if !confirmed {
		return `{"cancelled": true, "message": "User cancelled the operation"}`, nil
	}

	parentDir := filepath.Dir(destPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create destination directory: %w", err)
	}

	err = os.Rename(sourcePath, destPath)
	if err != nil {
		return "", fmt.Errorf("failed to move file: %w", err)
	}

	result := map[string]interface{}{
		"source":      source,
		"destination": destination,
		"success":     true,
		"message":     "File moved successfully",
	}

	jsonResult, _ := json.Marshal(result)
	return string(jsonResult), nil
}

// toolCopyFile copies a file
func (s *Session) toolCopyFile(args map[string]interface{}) (string, error) {
	source, ok := args["source"].(string)
	if !ok || source == "" {
		return "", fmt.Errorf("source argument is required")
	}

	destination, ok := args["destination"].(string)
	if !ok || destination == "" {
		return "", fmt.Errorf("destination argument is required")
	}

	sourcePath, err := s.resolvePath(source)
	if err != nil {
		return "", fmt.Errorf("invalid source path: %w", err)
	}

	destPath, err := s.resolvePath(destination)
	if err != nil {
		return "", fmt.Errorf("invalid destination path: %w", err)
	}

	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return "", fmt.Errorf("source file does not exist: %s", source)
	}

	destExists := false
	if _, err := os.Stat(destPath); err == nil {
		destExists = true
	}

	action := "Copy file"
	description := fmt.Sprintf("Source: %s\nDestination: %s", source, destination)
	if destExists {
		description += "\n⚠️  Destination file exists and will be overwritten!"
	}

	confirmed, err := s.confirmAction(action, description)
	if err != nil {
		return "", fmt.Errorf("failed to get confirmation: %w", err)
	}
	if !confirmed {
		return `{"cancelled": true, "message": "User cancelled the operation"}`, nil
	}

	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", fmt.Errorf("failed to read source file: %w", err)
	}

	parentDir := filepath.Dir(destPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create destination directory: %w", err)
	}

	err = os.WriteFile(destPath, data, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write destination file: %w", err)
	}

	result := map[string]interface{}{
		"source":      source,
		"destination": destination,
		"success":     true,
		"message":     "File copied successfully",
	}

	jsonResult, _ := json.Marshal(result)
	return string(jsonResult), nil
}

// toolFindFiles finds files by glob pattern
func (s *Session) toolFindFiles(args map[string]interface{}) (string, error) {
	pattern, ok := args["pattern"].(string)
	if !ok || pattern == "" {
		return "", fmt.Errorf("pattern argument is required")
	}

	searchPath := s.projectRoot
	if p, ok := args["path"].(string); ok && p != "" {
		resolved, err := s.resolvePath(p)
		if err != nil {
			return "", fmt.Errorf("invalid path: %w", err)
		}
		searchPath = resolved
	}

	var matches []string
	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		relPath, err := filepath.Rel(s.projectRoot, path)
		if err != nil {
			return nil
		}

		normalizedPath := strings.ReplaceAll(relPath, "\\", "/")
		if fsctx.ShouldIgnore(normalizedPath, s.cfg) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err == nil && matched && !info.IsDir() {
			matches = append(matches, normalizedPath)
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to search files: %w", err)
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

// toolFindFilesByExtension finds files by extension
func (s *Session) toolFindFilesByExtension(args map[string]interface{}) (string, error) {
	extension, ok := args["extension"].(string)
	if !ok || extension == "" {
		return "", fmt.Errorf("extension argument is required")
	}

	if !strings.HasPrefix(extension, ".") {
		extension = "." + extension
	}

	searchPath := s.projectRoot
	if p, ok := args["path"].(string); ok && p != "" {
		resolved, err := s.resolvePath(p)
		if err != nil {
			return "", fmt.Errorf("invalid path: %w", err)
		}
		searchPath = resolved
	}

	var matches []string
	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		relPath, err := filepath.Rel(s.projectRoot, path)
		if err != nil {
			return nil
		}

		normalizedPath := strings.ReplaceAll(relPath, "\\", "/")
		if fsctx.ShouldIgnore(normalizedPath, s.cfg) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !info.IsDir() && strings.EqualFold(filepath.Ext(path), extension) {
			matches = append(matches, normalizedPath)
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to search files: %w", err)
	}

	result := map[string]interface{}{
		"extension": extension,
		"path":      searchPath,
		"matches":   matches,
		"count":     len(matches),
	}

	jsonResult, _ := json.Marshal(result)
	return string(jsonResult), nil
}

// toolSearchSymbols searches for symbols in the indexed project
func (s *Session) toolSearchSymbols(args map[string]interface{}) (string, error) {
	symbol, ok := args["symbol"].(string)
	if !ok || symbol == "" {
		return "", fmt.Errorf("symbol argument is required")
	}

	if s.index == nil {
		return "", fmt.Errorf("project index not available")
	}

	searchPath := ""
	if p, ok := args["path"].(string); ok {
		searchPath = p
	}

	var results []map[string]interface{}

	// Search through all indexed files
	allPaths := s.index.GetAllFilePaths()
	for _, path := range allPaths {
		if searchPath != "" && !strings.HasPrefix(path, searchPath) {
			continue
		}

		fileInfo, ok := s.index.GetFileInfo(path)
		if !ok || fileInfo.IsDir {
			continue
		}

		for _, sym := range fileInfo.Symbols {
			if strings.EqualFold(sym.Name, symbol) {
				results = append(results, map[string]interface{}{
					"file":     path,
					"symbol":   sym.Name,
					"type":     sym.Type,
					"line":     sym.Line,
					"signature": sym.Signature,
				})
			}
		}
	}

	result := map[string]interface{}{
		"symbol":  symbol,
		"path":    searchPath,
		"matches": results,
		"count":   len(results),
	}

	jsonResult, _ := json.Marshal(result)
	return string(jsonResult), nil
}

// toolGetProjectStats returns project statistics
func (s *Session) toolGetProjectStats(args map[string]interface{}) (string, error) {
	if s.index == nil {
		return "", fmt.Errorf("project index not available")
	}

	var totalFiles int
	var totalDirs int
	var totalSize int64
	extensions := make(map[string]int)
	languages := make(map[string]int)
	var totalCodeLines int

	allPaths := s.index.GetAllFilePaths()
	for _, path := range allPaths {
		fileInfo, ok := s.index.GetFileInfo(path)
		if !ok {
			continue
		}
		if fileInfo.IsDir {
			totalDirs++
			continue
		}

		totalFiles++
		totalSize += fileInfo.Size

		ext := fileInfo.Extension
		if ext != "" {
			extensions[ext]++
		}

		codeExts := map[string]string{
			".go": "Go", ".php": "PHP", ".js": "JavaScript", ".ts": "TypeScript",
			".jsx": "JavaScript", ".tsx": "TypeScript", ".c": "C", ".cpp": "C++",
			".h": "C/C++", ".hpp": "C++", ".lua": "Lua", ".py": "Python",
			".java": "Java", ".rb": "Ruby", ".rs": "Rust", ".swift": "Swift",
			".kt": "Kotlin", ".scala": "Scala", ".cs": "C#", ".dart": "Dart",
			".sh": "Shell", ".bash": "Shell", ".zsh": "Shell",
		}

		if lang, ok := codeExts[ext]; ok {
			languages[lang]++
			fullPath, err := s.resolvePath(fileInfo.Path)
			if err == nil {
				if data, err := os.ReadFile(fullPath); err == nil {
					totalCodeLines += strings.Count(string(data), "\n") + 1
				}
			}
		}
	}

	result := map[string]interface{}{
		"files":         totalFiles,
		"directories":   totalDirs,
		"total_size":    totalSize,
		"code_lines":    totalCodeLines,
		"extensions":    extensions,
		"languages":     languages,
		"indexed_files": len(allPaths),
	}

	jsonResult, _ := json.Marshal(result)
	return string(jsonResult), nil
}

// toolGetFileInfo returns detailed file information
func (s *Session) toolGetFileInfo(args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("path argument is required")
	}

	fullPath, err := s.resolvePath(path)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		return "", fmt.Errorf("file does not exist: %w", err)
	}

	var lineCount int
	if !info.IsDir() {
		data, err := os.ReadFile(fullPath)
		if err == nil {
			lineCount = strings.Count(string(data), "\n") + 1
		}
	}

	result := map[string]interface{}{
		"path":          path,
		"name":          info.Name(),
		"size":          info.Size(),
		"is_dir":        info.IsDir(),
		"mode":          info.Mode().String(),
		"modified_time": info.ModTime().Format("2006-01-02 15:04:05"),
		"extension":     fsctx.GetFileExtension(path),
		"line_count":    lineCount,
	}

	if s.index != nil {
		if fileInfo, ok := s.index.GetFileInfo(path); ok {
			result["classes"] = fileInfo.Classes
			result["functions"] = fileInfo.Functions
			result["symbols_count"] = len(fileInfo.Symbols)
		}
	}

	jsonResult, _ := json.Marshal(result)
	return string(jsonResult), nil
}

// toolFindDependencies finds project dependencies
func (s *Session) toolFindDependencies(args map[string]interface{}) (string, error) {
	dependencyFiles := map[string]string{
		"package.json":    "npm/node",
		"go.mod":          "go",
		"composer.json":   "php/composer",
		"requirements.txt": "python/pip",
		"Cargo.toml":      "rust/cargo",
		"Gemfile":         "ruby/bundler",
		"pom.xml":         "java/maven",
		"build.gradle":    "java/gradle",
	}

	found := make(map[string]interface{})

	for fileName, manager := range dependencyFiles {
		filePath, err := s.resolvePath(fileName)
		if err != nil {
			continue // Skip if path is invalid (shouldn't happen for root-level files)
		}

		if _, err := os.Stat(filePath); err == nil {
			content, _, err := fsctx.ReadFile(s.projectRoot, fileName)
			if err == nil {
				lines := strings.Split(content, "\n")
				preview := lines
				if len(preview) > 10 {
					preview = preview[:10]
				}
				found[fileName] = map[string]interface{}{
					"manager":        manager,
					"path":           fileName,
					"content_preview": preview,
				}
			}
		}
	}

	result := map[string]interface{}{
		"dependency_files": found,
		"count":            len(found),
	}

	jsonResult, _ := json.Marshal(result)
	return string(jsonResult), nil
}

// toolGitStatus returns git repository status
func (s *Session) toolGitStatus(args map[string]interface{}) (string, error) {
	gitDir := filepath.Join(s.projectRoot, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return "", fmt.Errorf("not a git repository")
	}

	cmd := exec.Command("git", "-C", s.projectRoot, "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git status: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		lines = []string{}
	}

	var modified, added, deleted, untracked []string

	for _, line := range lines {
		if len(line) < 3 {
			continue
		}
		status := line[:2]
		file := strings.TrimSpace(line[2:])

		if status[0] == '?' {
			untracked = append(untracked, file)
		} else {
			if strings.Contains(status, "M") || strings.Contains(status, "m") {
				modified = append(modified, file)
			}
			if strings.Contains(status, "A") || strings.Contains(status, "a") {
				added = append(added, file)
			}
			if strings.Contains(status, "D") || strings.Contains(status, "d") {
				deleted = append(deleted, file)
			}
		}
	}

	result := map[string]interface{}{
		"modified":  modified,
		"added":     added,
		"deleted":   deleted,
		"untracked": untracked,
		"total":     len(modified) + len(added) + len(deleted) + len(untracked),
	}

	jsonResult, _ := json.Marshal(result)
	return string(jsonResult), nil
}

// toolGitDiff returns git diff for a file or directory
func (s *Session) toolGitDiff(args map[string]interface{}) (string, error) {
	gitDir := filepath.Join(s.projectRoot, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return "", fmt.Errorf("not a git repository")
	}

	path := ""
	if p, ok := args["path"].(string); ok {
		path = p
	}

	cmd := exec.Command("git", "-C", s.projectRoot, "diff", path)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git diff: %w", err)
	}

	diff := strings.TrimSpace(string(output))
	if diff == "" {
		diff = "No changes"
	}

	result := map[string]interface{}{
		"path": path,
		"diff": diff,
	}

	jsonResult, _ := json.Marshal(result)
	return string(jsonResult), nil
}

// toolFindSymbolReferences finds all references to a symbol
func (s *Session) toolFindSymbolReferences(args map[string]interface{}) (string, error) {
	symbol, ok := args["symbol"].(string)
	if !ok || symbol == "" {
		return "", fmt.Errorf("symbol argument is required")
	}

	searchPath := s.projectRoot
	if p, ok := args["path"].(string); ok && p != "" {
		resolved, err := s.resolvePath(p)
		if err != nil {
			return "", fmt.Errorf("invalid path: %w", err)
		}
		searchPath = resolved
	}

	var cmd *exec.Cmd
	if _, err := exec.LookPath("rg"); err == nil {
		cmd = exec.Command("rg", "-n", "--color", "never", "-w", symbol, searchPath)
	} else {
		cmd = exec.Command("grep", "-rn", "--color=never", "-w", symbol, searchPath)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		exitError, ok := err.(*exec.ExitError)
		if !ok || exitError.ExitCode() != 1 {
			return "", fmt.Errorf("search command failed: %w", err)
		}
		output = []byte{}
	}

	matches := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(matches) == 1 && matches[0] == "" {
		matches = []string{}
	}

	var results []map[string]interface{}
	for _, match := range matches {
		parts := strings.SplitN(match, ":", 2)
		if len(parts) == 2 {
			filePath := parts[0]
			rest := parts[1]

			lineParts := strings.SplitN(rest, ":", 2)
			if len(lineParts) == 2 {
				lineNum, _ := strconv.Atoi(lineParts[0])
				content := lineParts[1]

				relPath, _ := filepath.Rel(s.projectRoot, filePath)
				normalizedPath := strings.ReplaceAll(relPath, "\\", "/")

				results = append(results, map[string]interface{}{
					"file":    normalizedPath,
					"line":    lineNum,
					"content": strings.TrimSpace(content),
				})
			}
		}
	}

	result := map[string]interface{}{
		"symbol":  symbol,
		"path":    searchPath,
		"matches": results,
		"count":   len(results),
	}

	jsonResult, _ := json.Marshal(result)
	return string(jsonResult), nil
}

