package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/axon/pkg/logger"
)

// ChatWithTools sends a chat completion request with tools support
// It handles tool calls automatically and returns the final response
// Note: messages slice is modified in place to include all tool calls and responses
func (c *Client) ChatWithTools(ctx context.Context, messages []Message, tools []Tool, executeTool func(name string, args map[string]interface{}) (string, error)) (string, error) {
	maxIterations := 10 // Prevent infinite loops
	iteration := 0

	for iteration < maxIterations {
		// Create request with tools
		reqBody := ChatCompletionRequest{
			Model:       c.Model,
			Temperature: c.Temperature,
			Messages:    messages,
			Tools:       tools,
			ToolChoice:  "auto",
		}
		if c.MaxTokens > 0 {
			reqBody.MaxTokens = &c.MaxTokens
		}

		// Make API call
		response, err := c.chatCompletion(ctx, reqBody)
		if err != nil {
			return "", err
		}

		if len(response.Choices) == 0 {
			return "", fmt.Errorf("no choices in LLM response")
		}

		choice := response.Choices[0]
		assistantMsg := choice.Message

		// Add assistant message to history
		messages = append(messages, assistantMsg)

		// Check if model wants to call a tool
		if len(assistantMsg.ToolCalls) > 0 {
			// Execute all tool calls
			for _, toolCall := range assistantMsg.ToolCalls {
				// Log tool call
				logger.Logf("🔧 TOOL CALL RECEIVED: %s\n", toolCall.Function.Name)
				logger.Logf("   Tool Call ID: %s\n", toolCall.ID)
				logger.Logf("   Raw Arguments: %q\n", toolCall.Function.Arguments)

				// Parse arguments
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
					logger.Logf("   ❌ PARSE ERROR: %v\n", err)
					return "", fmt.Errorf("failed to parse tool arguments: %w (raw: %q)", err, toolCall.Function.Arguments)
				}

				logger.Logf("   Parsed Arguments: %+v\n", args)

				// Execute tool
				result, err := executeTool(toolCall.Function.Name, args)
				if err != nil {
					logger.Logf("   ❌ TOOL EXECUTION ERROR: %v\n", err)
					result = fmt.Sprintf("Error: %v", err)
				} else {
					logger.Logf("   ✅ TOOL RESULT: %s\n", logger.TruncateString(result, 200))
				}

				// Add tool result to messages
				messages = append(messages, Message{
					Role:       "tool",
					ToolCallID: toolCall.ID,
					Name:       toolCall.Function.Name,
					Content:    result,
				})
			}

			// Continue the loop to get the model's response with tool results
			iteration++
			continue
		}

		// Model returned a regular response (not a tool call)
		return assistantMsg.Content, nil
	}

	return "", fmt.Errorf("maximum iterations reached - possible infinite tool call loop")
}

// chatCompletion is the internal method that makes the HTTP request
func (c *Client) chatCompletion(ctx context.Context, reqBody ChatCompletionRequest) (*ChatCompletionResponse, error) {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v1/chat/completions", c.BaseURL)
	logger.LogRequest("POST", url, jsonData)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	logger.LogResponse(resp.StatusCode, string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LLM API returned status %d: %s", resp.StatusCode, string(body))
	}

	var completionResp ChatCompletionResponse
	if err := json.Unmarshal(body, &completionResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if completionResp.Error != nil {
		return nil, fmt.Errorf("LLM API error: %s", completionResp.Error.Message)
	}

	return &completionResp, nil
}
