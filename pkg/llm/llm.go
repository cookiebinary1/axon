package llm

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Message represents a chat message
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

// Client is the LLM API client
type Client struct {
	BaseURL     string
	Model       string
	Temperature float64
	MaxTokens   int
	HTTPClient  *http.Client
}

// NewClient creates a new LLM client with the given configuration
func NewClient(baseURL, model string, temperature float64) *Client {
	return &Client{
		BaseURL:     baseURL,
		Model:       model,
		Temperature: temperature,
		HTTPClient: &http.Client{
			Timeout: 120 * time.Second, // 2 minute timeout for LLM requests
		},
	}
}

// ChatCompletionRequest represents the request to the LLM API
type ChatCompletionRequest struct {
	Model       string    `json:"model"`
	Temperature float64   `json:"temperature"`
	Messages    []Message `json:"messages"`
	Tools       []Tool    `json:"tools,omitempty"`
	ToolChoice  string    `json:"tool_choice,omitempty"` // "auto", "none", or specific tool
	Stream      bool      `json:"stream,omitempty"`      // Enable streaming
	MaxTokens   *int      `json:"max_tokens,omitempty"`
}

// ChatCompletionResponse represents the response from the LLM API
type ChatCompletionResponse struct {
	Choices []struct {
		Message      Message `json:"message"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// GetSystemPrompt returns the system prompt for AXON
func GetSystemPrompt() string {
	return "You are AXON, a local code assistant running next to the user's codebase.\n" +
		"You specialize in PHP (Laravel), Go, JavaScript/TypeScript, shell, Docker, and Linux tooling.\n" +
		"You have access to tools that let you read files, list directories, search code, and modify files.\n" +
		"When you need to examine code, use the available tools instead of asking the user.\n" +
		"IMPORTANT: All write operations (write_file, create_file, update_file, string_replace, create_directory) require interactive user confirmation. The user will be prompted before any file or directory modification occurs.\n" +
		"You always respond with high-quality, concise code examples and short, focused explanations.\n" +
		"Prefer code blocks with proper language identifiers (```php, ```go, ```ts, etc.).\n" +
		"When given code from files, base your reasoning ONLY on this code and the described context. If you are missing information, use tools to read files before guessing.\n" +
		"When the question is about modifying code, describe the changes and show the final version or a clear patch-style diff."
}

// Chat sends a chat completion request to the LLM and returns the response
// This is a simple version without tools support
func (c *Client) Chat(ctx context.Context, messages []Message) (string, error) {
	reqBody := ChatCompletionRequest{
		Model:       c.Model,
		Temperature: c.Temperature,
		Messages:    messages,
	}
	if c.MaxTokens > 0 {
		reqBody.MaxTokens = &c.MaxTokens
	}

	completionResp, err := c.chatCompletion(ctx, reqBody)
	if err != nil {
		return "", err
	}

	if len(completionResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in LLM response")
	}

	content := completionResp.Choices[0].Message.Content
	if content == "" {
		// If content is empty, it might be a tool call - return error for now
		return "", fmt.Errorf("empty response from LLM (possibly tool call)")
	}

	return content, nil
}
