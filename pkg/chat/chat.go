package chat

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/axon/pkg/fsctx"
	"github.com/axon/pkg/indexer"
	"github.com/axon/pkg/llm"
	"github.com/axon/pkg/project"
	"github.com/charmbracelet/glamour"
)

// ANSI color codes
const (
	colorReset   = "\033[0m"
	colorCyan    = "\033[36m"
	colorMagenta = "\033[35m"
	colorYellow  = "\033[33m"
	colorGreen   = "\033[32m"
	colorRed     = "\033[31m"
	colorBlue    = "\033[34m"
	colorBold    = "\033[1m"
)

// Session represents an interactive chat session
type Session struct {
	client      *llm.Client
	projectRoot string
	cfg         *project.Config
	messages    []llm.Message
	debug       bool
	scanner     *bufio.Scanner // Scanner for user input (used for confirmations)
	index       *indexer.Index // Project index
}

// NewSession creates a new chat session
func NewSession(client *llm.Client, projectRoot string, cfg *project.Config, debug bool, projectIndex *indexer.Index) *Session {
	session := &Session{
		client:      client,
		projectRoot: projectRoot,
		cfg:         cfg,
		messages:    make([]llm.Message, 0),
		debug:       debug,
		scanner:     bufio.NewScanner(os.Stdin),
		index:       projectIndex,
	}

	// Add system message
	session.messages = append(session.messages, llm.Message{
		Role:    "system",
		Content: llm.GetSystemPrompt(),
	})

	return session
}

// Start starts the interactive chat session
func (s *Session) Start() error {
	// Print welcome message
	s.printWelcome()

	for {
		// Print prompt
		fmt.Printf("\n%sYou:%s ", colorCyan+colorBold, colorReset)

		// Read input
		if !s.scanner.Scan() {
			// EOF or error
			if err := s.scanner.Err(); err != nil {
				return fmt.Errorf("error reading input: %w", err)
			}
			break
		}

		input := strings.TrimSpace(s.scanner.Text())

		// Handle empty input
		if input == "" {
			continue
		}

		// Handle commands
		if strings.HasPrefix(input, "/") {
			if s.handleCommand(input) {
				continue
			}
			// If command handler returns false, treat it as regular input
		}

		// Add user message to history
		s.messages = append(s.messages, llm.Message{
			Role:    "user",
			Content: input,
		})

		// Show thinking indicator
		fmt.Printf("\n%sAXON is thinking...%s\n\n", colorYellow, colorReset)

		// Call LLM with tools support and streaming
		ctx := context.Background()
		tools := llm.GetAvailableTools()

		// Create tool executor
		executeTool := func(name string, args map[string]interface{}) (string, error) {
			return s.ExecuteTool(name, args)
		}

		// Store original message count
		originalCount := len(s.messages)

		// Buffer to accumulate markdown for real-time rendering
		var markdownBuffer strings.Builder
		var lastRenderedLines int
		var lastRenderLength int
		renderCounter := 0

		// Initialize markdown renderer once
		markdownRenderer, err := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(80),
		)
		if err != nil {
			// Fallback to basic rendering if glamour fails
			markdownRenderer = nil
		}

		// Streaming callback - render markdown in real-time
		firstToken := true
		streamCallback := func(chunk string) error {
			if firstToken {
				fmt.Printf("\n%sAXON:%s\n", colorMagenta+colorBold, colorReset)
				firstToken = false
				lastRenderedLines = 0
				lastRenderLength = 0
			}

			// Accumulate chunk
			markdownBuffer.WriteString(chunk)
			currentBuffer := markdownBuffer.String()
			currentLength := len(currentBuffer)

			// Render periodically or when structure changes (e.g., code block closes)
			shouldRender := false
			renderCounter++

			// Render if:
			// 1. Every 3 chunks (throttling for performance)
			// 2. Code block opens or closes (```)
			// 3. Significant length change (new paragraph/section)
			// 4. Newline character (likely end of sentence/paragraph)
			if renderCounter%3 == 0 {
				shouldRender = true
			} else if strings.Contains(chunk, "```") {
				shouldRender = true
			} else if strings.Contains(chunk, "\n") {
				shouldRender = true
			} else if currentLength-lastRenderLength > 50 {
				shouldRender = true
			}

			// Try to render the accumulated markdown
			if markdownRenderer != nil && shouldRender {
				s.renderStreamingMarkdown(markdownRenderer, currentBuffer, &lastRenderedLines)
				lastRenderLength = currentLength
			} else if markdownRenderer == nil {
				// Fallback: just print the chunk
				fmt.Print(chunk)
			}

			return nil
		}

		fullResponse, err := s.client.ChatWithToolsStream(ctx, s.messages, tools, executeTool, streamCallback)
		if err != nil {
			fmt.Printf("\n%sError:%s %v\n", colorRed+colorBold, colorReset, err)
			// Remove the last user message on error
			if len(s.messages) > originalCount {
				s.messages = s.messages[:originalCount]
			}
			continue
		}

		// Final render to ensure everything is properly formatted
		// Only re-render if we have a response but didn't render during streaming
		// (e.g., if markdownRenderer was nil during streaming but is now available)
		if fullResponse != "" && markdownRenderer != nil {
			// Check if we rendered during streaming (lastRenderedLines > 0 means we did)
			if lastRenderedLines == 0 {
				// We didn't render during streaming, so render now
				fmt.Printf("\n%sAXON:%s\n", colorMagenta+colorBold, colorReset)
				s.printFormattedMarkdown(fullResponse)
			} else {
				// We already rendered during streaming, just ensure newline at end
				fmt.Println()
			}
		} else if fullResponse != "" {
			// Fallback: just add newline
			fmt.Println()
		} else {
			// Add newline after streaming response
			fmt.Println()
		}
	}

	return nil
}

// handleCommand processes special commands starting with /
func (s *Session) handleCommand(input string) bool {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return false
	}

	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "/exit", "/quit", "/q":
		fmt.Printf("\n%sGoodbye!%s\n", colorBlue+colorBold, colorReset)
		// Exit will trigger deferred cleanup in main()
		os.Exit(0)
		return true
	case "/clear", "/reset":
		// Clear conversation history (keep system message)
		s.messages = []llm.Message{
			{Role: "system", Content: llm.GetSystemPrompt()},
		}
		fmt.Printf("\n%sConversation history cleared.%s\n", colorGreen, colorReset)
		return true
	case "/help", "/h":
		s.printHelp()
		return true
	case "/file":
		if len(args) == 0 {
			fmt.Printf("\n%sUsage:%s /file <path>\n", colorRed+colorBold, colorReset)
			return true
		}
		filePath := args[0]
		content, truncated, err := fsctx.ReadFile(s.projectRoot, filePath)
		if err != nil {
			fmt.Printf("\n%sError reading file:%s %v\n", colorRed+colorBold, colorReset, err)
			return true
		}
		lang := getLanguageFromExt(fsctx.GetFileExtension(filePath))
		fileNote := ""
		if truncated {
			fileNote = " (truncated to 200KB)"
		}
		fmt.Printf("\n%sFile:%s %s%s\n", colorBlue+colorBold, colorReset, filePath, fileNote)
		fmt.Printf("```%s\n%s\n```\n", lang, content)
		return true
	case "/explain":
		if len(args) == 0 {
			fmt.Printf("\n%sUsage:%s /explain <path> [start:end]\n", colorRed+colorBold, colorReset)
			return true
		}
		filePath := args[0]
		var content string
		var err error
		if len(args) > 1 {
			// Range specified
			lineRange := args[1]
			var startLine, endLine int
			if _, err := fmt.Sscanf(lineRange, "%d:%d", &startLine, &endLine); err != nil {
				fmt.Printf("\n%sInvalid range format:%s %s (expected start:end)\n", colorRed+colorBold, colorReset, lineRange)
				return true
			}
			content, err = fsctx.ReadFileRange(s.projectRoot, filePath, startLine, endLine)
		} else {
			var truncated bool
			content, truncated, err = fsctx.ReadFile(s.projectRoot, filePath)
			if truncated {
				fmt.Printf("\n%sFile truncated to first 200KB%s\n", colorYellow, colorReset)
			}
		}
		if err != nil {
			fmt.Printf("\n%sError reading file:%s %v\n", colorRed+colorBold, colorReset, err)
			return true
		}
		lang := getLanguageFromExt(fsctx.GetFileExtension(filePath))

		// Add explanation request to conversation
		explainPrompt := fmt.Sprintf("Explain the following code, focusing on what it does and potential issues:\n\nFile: %s\n```%s\n%s\n```", filePath, lang, content)
		s.messages = append(s.messages, llm.Message{
			Role:    "user",
			Content: explainPrompt,
		})

		fmt.Printf("\n%sAXON is thinking...%s\n\n", colorYellow, colorReset)
		ctx := context.Background()
		response, err := s.client.Chat(ctx, s.messages)
		if err != nil {
			fmt.Printf("%sError:%s %v\n", colorRed+colorBold, colorReset, err)
			s.messages = s.messages[:len(s.messages)-1]
			return true
		}

		s.messages = append(s.messages, llm.Message{
			Role:    "assistant",
			Content: response,
		})

		fmt.Printf("%sAXON:%s\n", colorMagenta+colorBold, colorReset)
		s.printFormattedMarkdown(response)
		return true
	default:
		// Unknown command, treat as regular input
		return false
	}
}

// printWelcome prints the welcome message
func (s *Session) printWelcome() {
	fmt.Printf("%s╔════════════════════════════════════════════════════════════╗%s\n", colorBold+colorBlue, colorReset)
	fmt.Printf("%s║                    AXON - Code Assistant                   ║%s\n", colorBold+colorBlue, colorReset)
	fmt.Printf("%s╚════════════════════════════════════════════════════════════╝%s\n", colorBold+colorBlue, colorReset)
	fmt.Printf("\n%sProject root:%s %s\n", colorBlue+colorBold, colorReset, s.projectRoot)
	fmt.Printf("%sLLM server:%s %s\n", colorBlue+colorBold, colorReset, s.cfg.LLM.BaseURL)
	fmt.Printf("%sModel:%s %s\n", colorBlue+colorBold, colorReset, s.cfg.LLM.Model)
	fmt.Printf("\n%sType your questions below. Use /help for commands.%s\n", colorYellow, colorReset)
	fmt.Println("   Press Ctrl+C or type /exit to quit.")
}

// printHelp prints help message
func (s *Session) printHelp() {
	fmt.Printf("\n%sAvailable commands:%s\n", colorBold+colorBlue, colorReset)
	fmt.Println("   /help, /h          - Show this help message")
	fmt.Println("   /clear, /reset     - Clear conversation history")
	fmt.Println("   /file <path>       - Display a file's contents")
	fmt.Println("   /explain <path>    - Explain code in a file")
	fmt.Println("   /explain <path> <start:end> - Explain a specific line range")
	fmt.Println("   /exit, /quit, /q   - Exit the chat")
	fmt.Printf("\n%sYou can also just type questions naturally!%s\n", colorYellow, colorReset)
	fmt.Println("   Example: \"How do I implement rate limiting in Laravel?\"")
}

// renderStreamingMarkdown renders markdown in real-time during streaming
// It clears previous output and re-renders the accumulated markdown
func (s *Session) renderStreamingMarkdown(renderer *glamour.TermRenderer, markdown string, lastLines *int) {
	// Try to render the markdown
	out, err := renderer.Render(markdown)
	if err != nil {
		// If rendering fails (e.g., incomplete markdown), skip this render
		// We'll try again on the next chunk
		return
	}

	// Count lines in the rendered output (including the header line with "AXON:")
	renderedLines := strings.Count(out, "\n")
	if renderedLines == 0 && out != "" {
		renderedLines = 1
	}

	// Clear previous rendering (move up and clear)
	if *lastLines > 0 {
		// Move cursor up by number of rendered lines
		fmt.Printf("\033[%dA", *lastLines)
		// Clear from cursor to end of screen
		fmt.Print("\033[J")
	}

	// Print the newly rendered markdown
	fmt.Print(out)

	// Update line count
	*lastLines = renderedLines
}

// printFormattedMarkdown renders and prints markdown with syntax highlighting
func (s *Session) printFormattedMarkdown(markdown string) {
	// Use glamour to render markdown with terminal-friendly styling
	// Auto-detect terminal style and width
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80), // Wrap at 80 characters (or use 0 for auto-detect)
	)
	if err != nil {
		// Fallback to basic printing if glamour fails
		fmt.Print(markdown)
		fmt.Println()
		return
	}

	out, err := r.Render(markdown)
	if err != nil {
		// Fallback to basic printing if rendering fails
		fmt.Print(markdown)
		fmt.Println()
		return
	}

	fmt.Print(out)
}

// printResponse prints the LLM response with formatting
// Handles code blocks specially (legacy function, kept for compatibility)
func (s *Session) printResponse(response string) {
	s.printFormattedMarkdown(response)
}

// getLanguageFromExt returns the language identifier for code blocks
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
