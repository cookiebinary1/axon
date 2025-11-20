package llm

// Tool represents a function/tool that the LLM can call
type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction describes a function that can be called
type ToolFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

// ToolCall represents a function call from the LLM
type ToolCall struct {
	Index    int              `json:"index"` // Index in the tool_calls array (for streaming)
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
}

// ToolCallFunction contains the function name and arguments
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// GetAvailableTools returns the list of tools available to the LLM
func GetAvailableTools() []Tool {
	return []Tool{
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "read_file",
				Description: "Read the contents of a file. Path is relative to project root.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Path to the file relative to project root",
						},
					},
					"required": []string{"path"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "list_directory",
				Description: "List files and directories in a directory. Path is relative to project root. If path is empty, lists project root.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Path to the directory relative to project root (empty string for project root)",
						},
					},
					"required": []string{},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "grep",
				Description: "Search for a pattern in files. Searches recursively from the given path (defaults to project root).",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"pattern": map[string]interface{}{
							"type":        "string",
							"description": "Search pattern (regular expression)",
						},
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Path to search from (relative to project root, empty string for project root)",
						},
					},
					"required": []string{"pattern"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "read_file_lines",
				Description: "Read specific lines from a file. Useful for reading a code section.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Path to the file relative to project root",
						},
						"start_line": map[string]interface{}{
							"type":        "integer",
							"description": "Starting line number (1-based)",
						},
						"end_line": map[string]interface{}{
							"type":        "integer",
							"description": "Ending line number (1-based, inclusive)",
						},
					},
					"required": []string{"path", "start_line", "end_line"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "write_file",
				Description: "Write content to a file. Path is relative to project root. Creates the file if it doesn't exist, overwrites if it does. Requires user confirmation.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Path to the file relative to project root",
						},
						"content": map[string]interface{}{
							"type":        "string",
							"description": "Content to write to the file",
						},
					},
					"required": []string{"path", "content"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "create_file",
				Description: "Create a new file with content. Path is relative to project root. Fails if file already exists. Requires user confirmation.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Path to the file relative to project root",
						},
						"content": map[string]interface{}{
							"type":        "string",
							"description": "Content to write to the file",
						},
					},
					"required": []string{"path", "content"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "update_file",
				Description: "Update an existing file by replacing its entire content. Path is relative to project root. Fails if file doesn't exist. Requires user confirmation.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Path to the file relative to project root",
						},
						"content": map[string]interface{}{
							"type":        "string",
							"description": "New content for the file",
						},
					},
					"required": []string{"path", "content"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "string_replace",
				Description: "Replace all occurrences of a string pattern in a file. Path is relative to project root. Requires user confirmation.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Path to the file relative to project root",
						},
						"old_string": map[string]interface{}{
							"type":        "string",
							"description": "String pattern to replace",
						},
						"new_string": map[string]interface{}{
							"type":        "string",
							"description": "Replacement string",
						},
					},
					"required": []string{"path", "old_string", "new_string"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "create_directory",
				Description: "Create a directory. Path is relative to project root. Creates parent directories if needed. Requires user confirmation.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Path to the directory relative to project root",
						},
					},
					"required": []string{"path"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "get_tree_list",
				Description: "Get a tree view of files and directories starting from a given path. Returns a hierarchical structure excluding ignored files/folders. Path is relative to project root. Use empty string for project root.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Path to the directory relative to project root (empty string for project root)",
						},
					},
					"required": []string{},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "get_file_symbols",
				Description: "Get a list of classes, functions, and other symbols from a code file. Path is relative to project root.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Path to the file relative to project root",
						},
					},
					"required": []string{"path"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "delete_file",
				Description: "Delete a file. Path is relative to project root. Requires user confirmation.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Path to the file relative to project root",
						},
					},
					"required": []string{"path"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "delete_directory",
				Description: "Delete a directory and all its contents. Path is relative to project root. Requires user confirmation.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Path to the directory relative to project root",
						},
					},
					"required": []string{"path"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "move_file",
				Description: "Move or rename a file. Paths are relative to project root. Requires user confirmation.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"source": map[string]interface{}{
							"type":        "string",
							"description": "Source path relative to project root",
						},
						"destination": map[string]interface{}{
							"type":        "string",
							"description": "Destination path relative to project root",
						},
					},
					"required": []string{"source", "destination"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "copy_file",
				Description: "Copy a file to a new location. Paths are relative to project root. Creates destination directory if needed. Requires user confirmation.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"source": map[string]interface{}{
							"type":        "string",
							"description": "Source path relative to project root",
						},
						"destination": map[string]interface{}{
							"type":        "string",
							"description": "Destination path relative to project root",
						},
					},
					"required": []string{"source", "destination"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "find_files",
				Description: "Find files by name pattern (glob pattern). Searches recursively from the given path (defaults to project root).",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"pattern": map[string]interface{}{
							"type":        "string",
							"description": "File name pattern (glob, e.g., '*.go', 'test_*.php')",
						},
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Path to search from (relative to project root, empty string for project root)",
						},
					},
					"required": []string{"pattern"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "find_files_by_extension",
				Description: "Find all files with a specific extension. Searches recursively from the given path (defaults to project root).",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"extension": map[string]interface{}{
							"type":        "string",
							"description": "File extension (e.g., '.go', '.php', '.js') - include the dot",
						},
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Path to search from (relative to project root, empty string for project root)",
						},
					},
					"required": []string{"extension"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "search_symbols",
				Description: "Search for a symbol (class, function, variable name) across the project. Returns all files where the symbol is defined or used.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"symbol": map[string]interface{}{
							"type":        "string",
							"description": "Symbol name to search for",
						},
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Path to search from (relative to project root, empty string for project root)",
						},
					},
					"required": []string{"symbol"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "get_project_stats",
				Description: "Get project statistics: file count, lines of code, languages used, etc.",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
					"required":   []string{},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "get_file_info",
				Description: "Get detailed information about a file: size, modification time, permissions, line count, etc.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Path to the file relative to project root",
						},
					},
					"required": []string{"path"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "find_dependencies",
				Description: "Find and list project dependencies from package.json, go.mod, composer.json, requirements.txt, Cargo.toml, etc.",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
					"required":   []string{},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "git_status",
				Description: "Get git repository status. Returns modified, added, deleted, and untracked files. Works only if project is a git repository.",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
					"required":   []string{},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "git_diff",
				Description: "Get git diff for a file or directory. Returns changes made. Works only if project is a git repository.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Path to the file or directory relative to project root (empty string for entire repository)",
						},
					},
					"required": []string{},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "find_symbol_references",
				Description: "Find all places where a symbol (class, function, variable) is used or referenced in the project. Searches code for occurrences of the symbol name.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"symbol": map[string]interface{}{
							"type":        "string",
							"description": "Symbol name to search for",
						},
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Path to search from (relative to project root, empty string for project root)",
						},
					},
					"required": []string{"symbol"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "execute",
				Description: "Execute a shell command in the project root directory. Returns stdout, stderr, and exit code. Requires user confirmation.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"command": map[string]interface{}{
							"type":        "string",
							"description": "Shell command to execute",
						},
						"description": map[string]interface{}{
							"type":        "string",
							"description": "Optional description of what this command does (for confirmation prompt)",
						},
					},
					"required": []string{"command"},
				},
			},
		},
	}
}
