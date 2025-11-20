# axon

**axon** is a lightweight local code assistant CLI tool powered by a local LLM server. It's designed to work within a project directory and provides code-focused assistance for PHP/Laravel, Go, JavaScript/TypeScript, Docker, and Linux tooling.

## Features

- **Interactive chat mode** - Chat with your code assistant in a conversational interface (like ChatGPT in CLI)
- **Ask questions** about your codebase with project context
- **Explain code** from specific files or line ranges using `/explain` command
- **View files** using `/file` command
- **Conversation history** - Maintain context throughout your session
- Project root detection (finds `.git` or `.axon.yml`)
- Configurable ignore patterns for large projects

## Prerequisites

axon requires **llama-server** (from llama.cpp) to be installed and available in your PATH.

### Installing llama-server

**macOS:**
```bash
# Using Homebrew
brew install llama.cpp

# Or build from source
git clone https://github.com/ggerganov/llama.cpp.git
cd llama.cpp
make
# Binary will be at ./server
```

**Linux:**
```bash
# Build from source
git clone https://github.com/ggerganov/llama.cpp.git
cd llama.cpp
make
# Binary will be at ./server
```

**Windows:**
```bash
# Using Git Bash or WSL
git clone https://github.com/ggerganov/llama.cpp.git
cd llama.cpp
# Use CMake to build (see llama.cpp README for details)
```

**Note:** axon can automatically start and stop llama-server for you (enabled by default). If you prefer to manage it manually, you can disable auto-start in the configuration.

## Installation

Build from source:

```bash
go build -o axon ./cmd/axon
```

Or install globally:

```bash
go install ./cmd/axon
```

## Configuration

### Project-level Config

Create a `.axon.yml` or `.axon.yaml` file in your project root:

```yaml
llm:
  base_url: "http://127.0.0.1:8080"
  model: "qwen2.5-coder-3b"
  temperature: 0.15

server:
  # Automatically start llama-server when axon starts
  # If server is not running, you'll be prompted to select a model
  auto_start: true
  # Path to llama-server binary (leave empty to use from PATH)
  server_path: ""
  # Default model (you can override via interactive selection)
  model: "Qwen/Qwen2.5-Coder-3B-Instruct-GGUF:Q4_K_M"

context:
  # Files/directories to ignore (glob patterns or simple prefixes)
  ignore:
    - "vendor/"
    - "node_modules/"
    - "storage/"
    - ".git/"
```

### Environment Variables

You can override configuration via environment variables:

- `AXON_LLM_BASE_URL` - LLM server base URL
- `AXON_LLM_MODEL` - Model identifier
- `AXON_LLM_TEMPERATURE` - Temperature (float)
- `AXON_SERVER_AUTO_START` - Enable/disable auto-start (set to `0` or `false` to disable)
- `AXON_SERVER_PATH` - Path to llama-server binary
- `AXON_SERVER_MODEL` - Model for llama-server
- `AXON_DEBUG=1` - Enable debug output to stderr
- `AXON_DEBUG_LOG=1` - Enable detailed logging to `.axon-debug.log` file

Environment variables take precedence over config file values.

## Usage

axon runs in **interactive chat mode** only. Simply run:

```bash
axon
```

This opens a conversational interface where you can:
- Chat naturally with the AI assistant
- Ask questions and get answers in a conversation format
- Use special commands for file operations
- Maintain conversation history throughout your session

### Interactive Commands

While in chat mode, you can use these commands:

- `/help` or `/h` - Show available commands
- `/clear` or `/reset` - Clear conversation history
- `/file <path>` - Display a file's contents
- `/explain <path>` - Explain code in a file
- `/explain <path> <start:end>` - Explain a specific line range
- `/exit`, `/quit`, or `/q` - Exit the chat

## How It Works

1. **Project Root Detection**: axon walks up from the current directory to find either:
   - A `.git` directory, or
   - A `.axon.yml`/`.axon.yaml` config file
   - If neither is found, the current directory is used as the project root

2. **Server Detection**: Checks if llama-server is already running at the configured URL

3. **Model Selection** (if server not running): If `auto_start` is enabled (default):
   - Prompts you to select a model family (Qwen2.5-Coder, DeepSeek Coder, CodeLlama, StarCoder)
   - Then prompts you to select a size variant (1.5B, 3B, 7B, etc.)
   - Automatically starts llama-server with your selected model
   - Waits for the server to be ready before starting chat

4. **Server Management**: 
   - Gracefully stops the server when you exit axon (if it started it)
   - If server was already running, leaves it running

5. **Configuration Loading**: Reads `.axon.yml` (if present) and applies environment variable overrides

6. **LLM Integration**: Sends requests to the local LLM server with:
   - System prompt defining AXON's role and expertise
   - User queries with optional file context
   - Proper code block formatting with language identifiers

## Examples

Start interactive chat:

```bash
axon
```

In chat mode, interact naturally:

```
╔════════════════════════════════════════════════════════════╗
║                    AXON - Code Assistant                   ║
╚════════════════════════════════════════════════════════════╝

📁 Project root: /path/to/project
🔗 LLM server: http://127.0.0.1:8080
🤖 Model: qwen2.5-coder-3b

💡 Type your questions below. Use /help for commands.
   Press Ctrl+C or type /exit to quit.

🤖 You: How do I implement rate limiting in Laravel?

💭 AXON is thinking...

💡 AXON: [response from LLM]

🤖 You: /explain app/Services/PaymentService.php

🤖 You: /file routes/web.php

🤖 You: /explain internal/server/http.go 120:180
```

## Debug Mode

Enable debug output to see project root, config loading, and request details:

```bash
AXON_DEBUG=1 axon
```

## Project Structure

```
axon/
├── cmd/axon/          # Main entrypoint
├── pkg/
│   ├── chat/          # Interactive chat session
│   ├── cli/           # CLI utilities (debug, etc.)
│   ├── llm/           # LLM API client
│   ├── project/       # Project root detection & config
│   └── fsctx/         # Filesystem helpers
├── README.md
└── go.mod
```

## Limitations

- **Read-only**: axon does not automatically modify files (safety first)
- **Local LLM required**: You must have llama-server running locally
- **File size limit**: Files larger than 200KB are truncated

## License

This project is provided as-is for local development use.

