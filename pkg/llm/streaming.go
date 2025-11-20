package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/axon/pkg/logger"
)

// ChatStreamCallback is called for each chunk received from the streaming API
type ChatStreamCallback func(chunk string) error

// ChatWithToolsStream sends a chat completion request with tools support and streaming
// It handles tool calls automatically and streams the final response
func (c *Client) ChatWithToolsStream(ctx context.Context, messages []Message, tools []Tool, executeTool func(name string, args map[string]interface{}) (string, error), callback ChatStreamCallback) (string, error) {
	maxIterations := 10 // Prevent infinite loops
	iteration := 0

	for iteration < maxIterations {
		// Create request with tools and streaming
		reqBody := ChatCompletionRequest{
			Model:       c.Model,
			Temperature: c.Temperature,
			Messages:    messages,
			Tools:       tools,
			ToolChoice:  "auto",
			Stream:      true, // Enable streaming
		}
		if c.MaxTokens > 0 {
			reqBody.MaxTokens = &c.MaxTokens
		}

		// Make streaming API call
		fullResponse, toolCalls, err := c.chatCompletionStream(ctx, reqBody, callback)
		if err != nil {
			return "", err
		}

		// Add assistant message to history
		assistantMsg := Message{
			Role:    "assistant",
			Content: fullResponse,
		}

		// Check if we got tool calls
		if len(toolCalls) > 0 {
			assistantMsg.ToolCalls = toolCalls
			messages = append(messages, assistantMsg)

			// Execute all tool calls
			for _, toolCall := range toolCalls {
				// Log tool call
				logger.Logf("🔧 TOOL CALL RECEIVED (streaming): %s\n", toolCall.Function.Name)
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
		messages = append(messages, assistantMsg)
		return fullResponse, nil
	}

	return "", fmt.Errorf("maximum iterations reached - possible infinite tool call loop")
}

// chatCompletionStream makes a streaming request to the LLM API
func (c *Client) chatCompletionStream(ctx context.Context, reqBody ChatCompletionRequest, callback ChatStreamCallback) (string, []ToolCall, error) {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v1/chat/completions", c.BaseURL)
	logger.LogRequest("POST", url, jsonData)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logger.LogResponse(resp.StatusCode, string(body))
		return "", nil, fmt.Errorf("LLM API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse Server-Sent Events (SSE) stream
	var fullResponse strings.Builder
	var toolCalls []ToolCall
	var finishReason string
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines and non-data lines
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}

		// Extract JSON data
		data := strings.TrimPrefix(line, "data: ")

		// Check for done signal
		if data == "[DONE]" {
			break
		}

		// Parse SSE JSON
		var streamResp struct {
			Choices []struct {
				Delta struct {
					Content   string     `json:"content"`
					ToolCalls []ToolCall `json:"tool_calls"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
		}

		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			// Log malformed JSON but skip it
			logger.Logf("⚠️  Malformed SSE JSON: %q, error: %v\n", data, err)
			continue
		}

		if len(streamResp.Choices) == 0 {
			continue
		}

		choice := streamResp.Choices[0]
		finishReason = choice.FinishReason

		// Handle content delta
		if choice.Delta.Content != "" {
			fullResponse.WriteString(choice.Delta.Content)
			// Call callback for streaming display
			if callback != nil {
				if err := callback(choice.Delta.Content); err != nil {
					return "", nil, err
				}
			}
		}

		// Handle tool calls - they come incrementally
		// Each chunk may contain part of a tool call
		// Tool calls in streaming format come with an "index" field to identify which tool call
		if len(choice.Delta.ToolCalls) > 0 {
			for _, deltaTC := range choice.Delta.ToolCalls {
				// Find existing tool call by index (if index is valid) or by ID
				var existingTC *ToolCall
				if deltaTC.Index >= 0 && deltaTC.Index < len(toolCalls) {
					// Check if index matches
					if toolCalls[deltaTC.Index].Index == deltaTC.Index {
						existingTC = &toolCalls[deltaTC.Index]
					}
				}

				// If not found by index, try to find by ID
				if existingTC == nil && deltaTC.ID != "" {
					for i := range toolCalls {
						if toolCalls[i].ID == deltaTC.ID {
							existingTC = &toolCalls[i]
							break
						}
					}
				}

				// Merge or create new tool call
				if existingTC != nil {
					// Merge: update fields that are present
					if deltaTC.ID != "" && existingTC.ID == "" {
						existingTC.ID = deltaTC.ID
					}
					if deltaTC.Type != "" && existingTC.Type == "" {
						existingTC.Type = deltaTC.Type
					}
					if deltaTC.Function.Name != "" && existingTC.Function.Name == "" {
						existingTC.Function.Name = deltaTC.Function.Name
					}
					// Accumulate arguments (they come as string chunks)
					if deltaTC.Function.Arguments != "" {
						existingTC.Function.Arguments += deltaTC.Function.Arguments
					}
				} else {
					// New tool call - ensure index is set
					if deltaTC.Index < 0 && len(toolCalls) > 0 {
						// If no index, use next available index
						deltaTC.Index = len(toolCalls)
					} else if deltaTC.Index < 0 {
						deltaTC.Index = 0
					}
					// Ensure we have enough capacity
					for len(toolCalls) <= deltaTC.Index {
						toolCalls = append(toolCalls, ToolCall{Index: len(toolCalls)})
					}
					// Place at correct index or append
					if deltaTC.Index < len(toolCalls) {
						toolCalls[deltaTC.Index] = deltaTC
						// Ensure index is set correctly
						if toolCalls[deltaTC.Index].Index != deltaTC.Index {
							toolCalls[deltaTC.Index].Index = deltaTC.Index
						}
					} else {
						toolCalls = append(toolCalls, deltaTC)
					}
				}
			}
		}

		// Check finish reason - if tool_calls, tool calls are complete
		if finishReason == "tool_calls" {
			logger.Logf("✅ Tool calls complete (finish_reason=tool_calls), total tool calls: %d\n", len(toolCalls))
			// Filter out any incomplete tool calls (those without ID or function name)
			completeToolCalls := make([]ToolCall, 0, len(toolCalls))
			for _, tc := range toolCalls {
				if tc.ID != "" && tc.Function.Name != "" {
					completeToolCalls = append(completeToolCalls, tc)
				} else {
					logger.Logf("   ⚠️  Skipping incomplete tool call: index=%d, id=%q, name=%q\n",
						tc.Index, tc.ID, tc.Function.Name)
				}
			}
			toolCalls = completeToolCalls
		}
	}

	if err := scanner.Err(); err != nil {
		return "", nil, fmt.Errorf("error reading stream: %w", err)
	}

	// Filter out incomplete tool calls before returning
	// Only return tool calls if finish_reason was "tool_calls"
	if finishReason == "tool_calls" {
		completeToolCalls := make([]ToolCall, 0, len(toolCalls))
		for _, tc := range toolCalls {
			if tc.ID != "" && tc.Function.Name != "" {
				completeToolCalls = append(completeToolCalls, tc)
			}
		}
		// Log final response summary
		logger.Logf("✅ STREAMING COMPLETE: finish_reason=tool_calls, tool_calls_count=%d\n", len(completeToolCalls))
		return fullResponse.String(), completeToolCalls, nil
	}

	// Log final response summary for regular responses
	if fullResponse.Len() > 0 {
		logger.Logf("✅ STREAMING COMPLETE: finish_reason=%s, response_length=%d\n", finishReason, fullResponse.Len())
	}
	return fullResponse.String(), nil, nil
}
