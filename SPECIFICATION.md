You are an expert Go engineer and tooling architect.
Your task is to design and implement a **CLI code assistant** called **axon**.

The goal: axon should behave like a lightweight local “Codex / Cloud Code” for working inside a project folder, powered by a local LLM server (llama-server with Qwen2.5-Coder-3B-Instruct-GGUF:Q4_K_M).

The user is an experienced backend developer (PHP/Laravel, Go, JS/TS, Linux, Docker), comfortable with the terminal. They want a practical, robust tool, not a toy demo.

==================================================
1. HIGH-LEVEL OVERVIEW
==================================================

Build a CLI tool:

- Name: `axon`
- Language: **Go**
- Purpose: local code assistant for a project directory, backed by a local LLM API.
- Core capabilities:
  - Ask the model coding questions with project context.
  - Read and analyze files in the current project.
  - Explain, summarize, and refactor code.
  - Perform simple search / “grep-like” operations and optionally let the model reason over the results.
- The tool should NOT be a generic chat bot; it is **code/project-focused**.

The LLM server is provided externally; axon just calls it.

==================================================
2. LLM BACKEND INTEGRATION
==================================================

The LLM backend is `llama-server` started like:

  llama-server -hf Qwen/Qwen2.5-Coder-3B-Instruct-GGUF:Q4_K_M

Assume it exposes an **OpenAI-compatible** Chat Completions API:

- Base URL (default): `http://127.0.0.1:8080`
- Endpoint: `POST /v1/chat/completions`
- Request JSON shape:

  {
    "model": "<string>",
    "temperature": <float>,
    "messages": [
      { "role": "system" | "user" | "assistant", "content": "<text>" },
      ...
    ]
  }

- You can choose a sensible model identifier string (e.g. `"qwen2.5-coder-3b"`); llama-server will ignore it anyway.

Implement a small Go client package, e.g. `pkg/llm`, with:

- Config:
  - BaseURL string (default `http://127.0.0.1:8080`)
  - Model   string (default `qwen2.5-coder-3b`)
  - Temperature float (e.g. 0.15)
  - MaxTokens (optional, if supported)
- Function(s) like:

  - `Chat(ctx context.Context, messages []Message) (string, error)`
  - Optional: streaming API (chunked responses); if streaming is extra work, implement non-streaming first, but keep the code structured so that streaming can be added later.

- Handle:
  - HTTP errors, timeouts, network failures.
  - Invalid JSON responses.
  - Return clear Go errors.

Messages model:

- Small struct:

  type Message struct {
      Role    string `json:"role"`
      Content string `json:"content"`
  }

==================================================
3. CONFIGURATION & PROJECT ROOT DETECTION
==================================================

axon should work “inside a project folder” and find the project root.

Rules:

1. Project root detection:
   - Start from current working directory.
   - Walk upwards until you find either:
     - `.git` directory, OR
     - `.axon.yml` / `.axon.yaml` config file.
   - If neither is found, treat the starting directory as the project root.
   - Expose a function: `FindProjectRoot(startDir string) (string, error)` in a separate package, e.g. `pkg/project`.

2. Configuration:
   - Optional project-level config file: `.axon.yml` in the project root.
   - Structure (YAML), e.g.:

     ```yaml
     llm:
       base_url: "http://127.0.0.1:8080"
       model: "qwen2.5-coder-3b"
       temperature: 0.15

     context:
       # files / directories to ignore (glob or regex is fine)
       ignore:
         - "vendor/"
         - "node_modules/"
         - "storage/"
         - ".git/"
     ```

   - Also read environment variables:

     - `AXON_LLM_BASE_URL`
     - `AXON_LLM_MODEL`
     - `AXON_LLM_TEMPERATURE`

   - Precedence: environment variables override config file, config file overrides defaults.

==================================================
4. CLI DESIGN & UX
==================================================

Use Go standard library plus one CLI helper (e.g. Cobra, urfave/cli, or your own small parser). Keep dependencies reasonable.

Top-level binary: `axon`
Subcommands:

1) `axon ask "<question>"`

   - One-shot question/answer.
   - Example usage:
     - `axon ask "Explain what the HTTP middleware in app/Http/Middleware/CheckRole.php does."`
     - `axon ask "Design a simple Go http server with /health endpoint."`
   - Behavior:
     - Detect project root.
     - Build a system prompt that defines AXON’s role (see System Prompt section).
     - Build messages:
       - `system`: role description + mention of languages (PHP/Laravel, Go, JS/TS, Docker, Linux).
       - `user`: content = the question.
     - Optionally support flags:
       - `--file path/to/file` : include the content of the file (or a truncated version) in the prompt.
       - `--with-context` : automatically sample a small set of relevant files (optional, if easy; could be future work, but design the code so it’s easy to extend).

2) `axon explain <path> [--range start:end]`

   - Explain a specific file or snippet.
   - Examples:
     - `axon explain app/Http/Controllers/UserController.php`
     - `axon explain internal/server/http.go --range 120:180`
   - Behavior:
     - Read the file from the project root.
     - If `--range` is provided, include only that line range in the prompt.
     - Send a prompt like:
       - system: “You are AXON, a senior engineer…”
       - user: “Explain the following code, focusing on what it does and potential issues. Code: ```<code>```”

3) `axon search "<pattern>" [path]`

   - Thin wrapper over `rg` (ripgrep) or `grep -rn`, plus optional LLM reasoning.
   - Behavior:
     - Execute `rg` if available, otherwise fallback to `grep -Rn`.
     - Print matches directly to stdout.
     - Optional flag `--explain`:
       - When set, also send a summarized version of the grep results to the LLM and ask for:
         - “Where is the main place to modify this behavior?”
         - “Which files are important for <pattern>?”
   - This demonstrates simple command execution + LLM reasoning over the output.

4) (Optional/Future) `axon edit` – not necessary in the first version, but structure the code so it’s possible to add later:
   - e.g. `axon edit path/to/file.php --instruction "Refactor to use dependency injection"` and print a patch.

For now, implement at least:

- `axon ask`
- `axon explain`
- `axon search`

All subcommands must:

- Respect the project root and ignore lists from config.
- Show clear error messages when files are missing, project root not found, or the LLM server is not reachable.

==================================================
5. SYSTEM PROMPT DESIGN (FOR LLM)
==================================================

Implement a helper in `pkg/llm` or similar to build the system prompt.

The system prompt should:

- Name the assistant: **AXON**
- Role: local code assistant for a single project.
- Emphasize:
  - Focus on PHP (especially Laravel), Go, JavaScript/TypeScript, shell scripts, Docker, and Linux tooling.
  - Provide concise, high-quality code with short explanations.
  - Prefer code blocks with correct language identifiers (```php, ```go, ```ts, etc.).
  - When working with code from files, always respect the given context and avoid hallucinating non-existent functions/files.
  - If information is missing, explicitly state what is missing.

Example system message (you can refine wording):

> You are AXON, a local code assistant running next to the user’s codebase.
> You specialize in PHP (Laravel), Go, JavaScript/TypeScript, shell, Docker, and Linux tooling.
> You always respond with high-quality, concise code examples and short, focused explanations.
> Prefer code blocks with proper language identifiers.
> When given code from files, base your reasoning ONLY on this code and the described context. If you are missing information, say so explicitly instead of guessing.
> When the question is about modifying code, describe the changes and show the final version or a clear patch-style diff.

Make sure this system message is reused across subcommands in a central place.

==================================================
6. FILE & DIRECTORY HANDLING
==================================================

Implement a small `pkg/fsctx` or similar to manage:

- Resolving paths relative to the project root.
- Reading files safely:
  - Check file size; if it’s too large (e.g. > 200 KB), truncate and mention in the prompt that it is truncated.
- Respect ignore list from config (e.g. skip `vendor/`, `node_modules/`, `storage/`, `.git/`).
- Optionally (if easy): Utility to collect a small set of “context” files to send to the LLM based on file extension or a simple heuristic.

==================================================
7. IMPLEMENTATION DETAILS & CODE QUALITY
==================================================

General:

- Language: Go
- Use Go modules.
- Use idiomatic package structure, for example:

  - `cmd/axon/main.go`      – main entrypoint using a CLI framework or manual parsing.
  - `pkg/cli`               – subcommand definitions / wiring.
  - `pkg/llm`               – LLM API client and system prompt builder.
  - `pkg/project`           – project root detection and config loading.
  - `pkg/fsctx`             – filesystem helpers for reading files with ignore rules.

- Comments in code: **in English** (important).
- Keep the code reasonably small and focused, but not hacky.

Error handling:

- Always handle errors explicitly.
- For CLI UX, print human-friendly error messages to stderr and exit with non-zero status on failures.

Logging & debug:

- Add a `--debug` global flag (or `AXON_DEBUG=1` env var) to print extra information:
  - Which project root was detected.
  - Which config file was loaded.
  - Brief information about the LLM request (without dumping full text by default).

==================================================
8. TESTS & DOCUMENTATION
==================================================

Tests:

- Add unit tests for:
  - Project root detection.
  - Config loading and environment override.
  - Basic LLM client behavior (mock the HTTP server).

Documentation:

- Add a `README.md` at the root describing:
  - What axon is.
  - How to install (e.g. `go build -o axon ./cmd/axon`).
  - How to configure `.axon.yml`.
  - How to run llama-server with Qwen2.5-Coder-3B-Instruct-GGUF:Q4_K_M.
  - Example commands:

    - `axon ask "How do I create a Laravel job that sends emails?"`
    - `axon explain app/Http/Middleware/CheckRole.php`
    - `axon search "DB::table" app/ --explain`

==================================================
9. NON-GOALS / SAFETY
==================================================

- axon should NOT execute arbitrary shell commands from the LLM.
  - `axon search` may internally use `rg` or `grep` to search code, but this is **driven by the CLI**, not by free-form instructions from the LLM.
- Do not implement automatic file modifications yet (no auto-editing). The first version is read-only with suggestions and explanations.
- If the user asks axon to “run dangerous commands” (e.g. rm -rf), the LLM can suggest but the tool itself must NOT run them.

==================================================
10. WHAT TO DELIVER
==================================================

Deliver:

1. Complete Go source tree with:
   - `cmd/axon/main.go`
   - Implementation of `axon ask`, `axon explain`, `axon search`.
   - Packages: `llm`, `project`, `fsctx`, etc., as described.
2. A minimal but clear `README.md`.
3. Example `.axon.yml`.
4. Basic unit tests for key components.

Make sure the final solution is **compilable** and works on macOS/Linux with Go installed, assuming the user has llama-server running locally at `http://127.0.0.1:8080` with the Qwen2.5-Coder-3B-Instruct-GGUF:Q4_K_M model.

Once you have the full design in mind, implement the code directly, without pseudo-code.
