package chat

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/axon/pkg/fsctx"
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
}

// NewSession creates a new chat session
func NewSession(client *llm.Client, projectRoot string, cfg *project.Config, debug bool) *Session {
	session := &Session{
		client:      client,
		projectRoot: projectRoot,
		cfg:         cfg,
		messages:    make([]llm.Message, 0),
		debug:       debug,
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

	scanner := bufio.NewScanner(os.Stdin)

	for {
		// Print prompt
		fmt.Printf("\n%sYou:%s ", colorCyan+colorBold, colorReset)

		// Read input
		if !scanner.Scan() {
			// EOF or error
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("error reading input: %w", err)
			}
			break
		}

		input := strings.TrimSpace(scanner.Text())

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

		// Streaming callback - print tokens as they arrive
		firstToken := true
		streamCallback := func(chunk string) error {
			if firstToken {
				fmt.Printf("\n%sAXON:%s ", colorMagenta+colorBold, colorReset)
				firstToken = false
			}
			// Print raw chunk for immediate feedback during streaming
			fmt.Print(chunk)
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

		// Render markdown-formatted response
		if fullResponse != "" {
			// Clear the last line (where "AXON:" was printed) and render formatted markdown
			fmt.Print("\r\033[K") // Clear current line
			fmt.Printf("%sAXON:%s\n", colorMagenta+colorBold, colorReset)
			s.printFormattedMarkdown(fullResponse)
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
