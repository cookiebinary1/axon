package main

import (
	"fmt"
	"os"

	"github.com/axon/pkg/chat"
	"github.com/axon/pkg/cli"
	"github.com/axon/pkg/llm"
	"github.com/axon/pkg/logger"
	"github.com/axon/pkg/project"
	"github.com/axon/pkg/server"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorBlue   = "\033[34m"
	colorBold   = "\033[1m"
)

func main() {
	// Parse debug flag from environment
	cli.Debug = os.Getenv("AXON_DEBUG") == "1"

	// Handle help flag
	if len(os.Args) > 1 {
		arg := os.Args[1]
		if arg == "help" || arg == "--help" || arg == "-h" {
			printUsage()
			return
		}
		fmt.Fprintf(os.Stderr, "axon: interactive chat mode\n")
		fmt.Fprintf(os.Stderr, "Run 'axon' or 'axon --help' for more information.\n")
		os.Exit(1)
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to get current directory: %v\n", err)
		os.Exit(1)
	}

	// Find project root
	projectRoot, err := project.FindProjectRoot(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to find project root: %v\n", err)
		os.Exit(1)
	}

	// Initialize debug logger
	if err := logger.InitLogger(projectRoot); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to initialize debug logger: %v\n", err)
	}
	defer logger.CloseLogger()

	// Load configuration
	cfg, err := project.LoadConfig(projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load config: %v\n", err)
		os.Exit(1)
	}

	cli.Debugf("Loaded config from project root: %s", projectRoot)

	// Check if server is already running
	var srv *server.Server
	if server.CheckRunning(cfg.LLM.BaseURL) {
		fmt.Fprintf(os.Stderr, "%sLLM server is already running at %s%s\n", colorGreen+colorBold, cfg.LLM.BaseURL, colorReset)
	} else if cfg.Server.AutoStart {
		// Server is not running, ask user to select model
		selectedModel, err := server.SelectModel()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sError selecting model:%s %v\n", colorRed+colorBold, colorReset, err)
			os.Exit(1)
		}

		// Create server with selected model
		srv = server.NewServer(cfg.Server.ServerPath, cfg.LLM.BaseURL, selectedModel, cli.Debug)

		// Setup signal handling to stop server on exit
		srv.SetupSignalHandling()

		// Start the server
		if err := srv.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "%sError starting LLM server:%s %v\n", colorRed+colorBold, colorReset, err)
			fmt.Fprintf(os.Stderr, "%sTip:%s You can disable auto-start by setting AXON_SERVER_AUTO_START=0\n", colorYellow, colorReset)
			fmt.Fprintf(os.Stderr, "   or configure a manual server path in .axon.yml\n")
			os.Exit(1)
		}

		// Ensure server is stopped on exit
		defer func() {
			if srv != nil {
				srv.Stop()
			}
		}()
	} else {
		fmt.Fprintf(os.Stderr, "%sLLM server is not running and auto-start is disabled.%s\n", colorYellow+colorBold, colorReset)
		fmt.Fprintf(os.Stderr, "   Please start llama-server manually or enable auto-start in config.\n")
		os.Exit(1)
	}

	// Start interactive chat mode
	startInteractiveMode(projectRoot, cfg, srv)
}

func startInteractiveMode(projectRoot string, cfg *project.Config, srv *server.Server) {
	// Create LLM client
	client := llm.NewClient(cfg.LLM.BaseURL, cfg.LLM.Model, cfg.LLM.Temperature)

	// Create chat session with server reference for cleanup
	session := chat.NewSession(client, projectRoot, cfg, cli.Debug)

	// Start interactive chat
	err := session.Start()

	// Stop server on chat exit
	if srv != nil {
		srv.Stop()
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	usage := `axon - Interactive code assistant powered by a local LLM

USAGE:
    axon                    Start interactive chat mode
    axon --help             Show this help message

INTERACTIVE CHAT MODE:
    axon is an interactive chat-based code assistant. Simply run 'axon' to start.

    You can:
    - Chat naturally with the AI assistant
    - Ask questions about your codebase
    - Use special commands for file operations

COMMANDS IN CHAT:
    /help, /h               Show available commands
    /clear, /reset          Clear conversation history
    /file <path>            Display a file's contents
    /explain <path>         Explain code in a file
    /explain <path> <start:end>  Explain a specific line range
    /exit, /quit, /q        Exit the chat

EXAMPLES:
    axon                                    # Start interactive chat
    
    In chat mode:
    You: How do I create a Laravel job that sends emails?
    You: /explain app/Http/Middleware/CheckRole.php
    You: /file routes/web.php
    You: /explain internal/server/http.go 120:180

CONFIGURATION:
    Configuration can be set via:
    - .axon.yml or .axon.yaml in project root
    - Environment variables (AXON_LLM_BASE_URL, AXON_LLM_MODEL, AXON_LLM_TEMPERATURE)

DEBUG:
    Set AXON_DEBUG=1 to enable debug output

For more information, see README.md
`
	fmt.Print(usage)
}
