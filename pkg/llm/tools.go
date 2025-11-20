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
	}
}
