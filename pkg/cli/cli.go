package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/axon/pkg/fsctx"
	"github.com/axon/pkg/llm"
	"github.com/axon/pkg/project"
)

// Debug mode flag
var Debug bool

// Debugf prints a debug message if debug mode is enabled
func Debugf(format string, args ...interface{}) {
	if Debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

// HandleAsk handles the "ask" subcommand
func HandleAsk(question string, filePath string, withContext bool, projectRoot string, cfg *project.Config) error {
	Debugf("Project root: %s", projectRoot)

	// Create LLM client
	client := llm.NewClient(cfg.LLM.BaseURL, cfg.LLM.Model, cfg.LLM.Temperature)

	// Build messages
	messages := []llm.Message{
		{Role: "system", Content: llm.GetSystemPrompt()},
	}

	// Build user message
	userContent := question

	// Optionally include file content
	if filePath != "" {
		content, truncated, err := fsctx.ReadFile(projectRoot, filePath)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		lang := getLanguageFromExt(fsctx.GetFileExtension(filePath))
		fileNote := ""
		if truncated {
			fileNote = " (Note: File was truncated to first 200KB)\n"
		}
		userContent = fmt.Sprintf("%s\n\nFile: %s%s\n```%s\n%s\n```", question, filePath, fileNote, lang, content)
	}

	messages = append(messages, llm.Message{
		Role:    "user",
		Content: userContent,
	})

	Debugf("Sending request to LLM at %s", cfg.LLM.BaseURL)

	// Call LLM
	ctx := context.Background()
	response, err := client.Chat(ctx, messages)
	if err != nil {
		return fmt.Errorf("LLM request failed: %w", err)
	}

	// Print response
	fmt.Println(response)
	return nil
}

// HandleExplain handles the "explain" subcommand
func HandleExplain(filePath string, lineRange string, projectRoot string, cfg *project.Config) error {
	Debugf("Project root: %s", projectRoot)

	// Create LLM client
	client := llm.NewClient(cfg.LLM.BaseURL, cfg.LLM.Model, cfg.LLM.Temperature)

	// Read file content
	var content string
	var truncated bool
	var err error

	if lineRange != "" {
		// Parse range (format: start:end)
		var startLine, endLine int
		if _, err := fmt.Sscanf(lineRange, "%d:%d", &startLine, &endLine); err != nil {
			return fmt.Errorf("invalid range format: %s (expected start:end)", lineRange)
		}

		content, err = fsctx.ReadFileRange(projectRoot, filePath, startLine, endLine)
		if err != nil {
			return fmt.Errorf("failed to read file range: %w", err)
		}
		truncated = false // Range reads are always full, no truncation
	} else {
		content, truncated, err = fsctx.ReadFile(projectRoot, filePath)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
	}

	lang := getLanguageFromExt(fsctx.GetFileExtension(filePath))
	fileNote := ""
	if truncated {
		fileNote = " (Note: File was truncated to first 200KB)"
	}

	rangeNote := ""
	if lineRange != "" {
		rangeNote = fmt.Sprintf(" (Lines %s)", lineRange)
	}

	// Build messages
	messages := []llm.Message{
		{Role: "system", Content: llm.GetSystemPrompt()},
		{
			Role: "user",
			Content: fmt.Sprintf(
				"Explain the following code, focusing on what it does and potential issues.%s%s\n\nFile: %s\n```%s\n%s\n```",
				fileNote,
				rangeNote,
				filePath,
				lang,
				content,
			),
		},
	}

	Debugf("Sending request to LLM at %s", cfg.LLM.BaseURL)

	// Call LLM
	ctx := context.Background()
	response, err := client.Chat(ctx, messages)
	if err != nil {
		return fmt.Errorf("LLM request failed: %w", err)
	}

	// Print response
	fmt.Println(response)
	return nil
}

// HandleSearch handles the "search" subcommand
func HandleSearch(pattern string, searchPath string, explain bool, projectRoot string, cfg *project.Config) error {
	Debugf("Project root: %s", projectRoot)

	// Determine search directory
	if searchPath == "" {
		searchPath = projectRoot
	} else {
		searchPath = fsctx.ResolvePath(projectRoot, searchPath)
	}

	// Execute search command
	var cmd *exec.Cmd
	var searchCmd string

	// Try ripgrep first
	if _, err := exec.LookPath("rg"); err == nil {
		searchCmd = "rg"
		cmd = exec.Command("rg", "-n", "--color", "never", pattern, searchPath)
	} else {
		// Fallback to grep
		searchCmd = "grep"
		cmd = exec.Command("grep", "-rn", "--color=never", pattern, searchPath)
	}

	Debugf("Running %s: %s", searchCmd, strings.Join(cmd.Args, " "))

	output, err := cmd.CombinedOutput()
	if err != nil {
		// grep/rg exit code 1 means no matches found, which is fine
		exitError, ok := err.(*exec.ExitError)
		if !ok || exitError.ExitCode() != 1 {
			return fmt.Errorf("search command failed: %w", err)
		}
	}

	// Print search results directly
	if len(output) > 0 {
		fmt.Print(string(output))
	} else {
		fmt.Fprintf(os.Stderr, "No matches found for pattern: %s\n", pattern)
	}

	// Optionally explain with LLM
	if explain && len(output) > 0 {
		fmt.Fprintln(os.Stderr, "\n--- LLM Analysis ---")

		// Create LLM client
		client := llm.NewClient(cfg.LLM.BaseURL, cfg.LLM.Model, cfg.LLM.Temperature)

		// Prepare summary for LLM (limit size)
		outputStr := string(output)
		if len(outputStr) > 5000 {
			outputStr = outputStr[:5000] + "\n... (truncated)"
		}

		messages := []llm.Message{
			{Role: "system", Content: llm.GetSystemPrompt()},
			{
				Role: "user",
				Content: fmt.Sprintf(
					"Analyze the following search results for pattern '%s':\n\n```\n%s\n```\n\nQuestions:\n1. Where is the main place to modify this behavior?\n2. Which files are important for '%s'?",
					pattern,
					outputStr,
					pattern,
				),
			},
		}

		Debugf("Sending search analysis request to LLM at %s", cfg.LLM.BaseURL)

		ctx := context.Background()
		response, err := client.Chat(ctx, messages)
		if err != nil {
			return fmt.Errorf("LLM analysis failed: %w", err)
		}

		fmt.Println(response)
	}

	return nil
}

// getLanguageFromExt returns the language identifier for code blocks based on file extension
func getLanguageFromExt(ext string) string {
	extMap := map[string]string{
		".go":         "go",
		".php":        "php",
		".js":         "javascript",
		".ts":         "typescript",
		".jsx":        "javascript",
		".tsx":        "typescript",
		".sh":         "bash",
		".bash":       "bash",
		".yml":        "yaml",
		".yaml":       "yaml",
		".json":       "json",
		".md":         "markdown",
		".sql":        "sql",
		".html":       "html",
		".css":        "css",
		".dockerfile": "dockerfile",
		".rb":         "ruby",
		".py":         "python",
		".java":       "java",
		".c":          "c",
		".cpp":        "cpp",
		".h":          "c",
		".hpp":        "cpp",
	}

	if lang, ok := extMap[ext]; ok {
		return lang
	}
	return "text"
}
