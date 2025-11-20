package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_Chat(t *testing.T) {
	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("Expected /v1/chat/completions, got %s", r.URL.Path)
		}

		// Verify request body
		var req ChatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		// Send mock response
		response := ChatCompletionResponse{
			Choices: []struct {
				Message      Message `json:"message"`
				FinishReason string  `json:"finish_reason"`
			}{
				{
					Message: Message{
						Role:    "assistant",
						Content: "This is a test response",
					},
					FinishReason: "stop",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client with test server URL
	client := NewClient(server.URL, "test-model", 0.7)

	// Test Chat
	messages := []Message{
		{Role: "system", Content: "You are a helpful assistant"},
		{Role: "user", Content: "Hello"},
	}

	ctx := context.Background()
	response, err := client.Chat(ctx, messages)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if response != "This is a test response" {
		t.Errorf("Expected 'This is a test response', got %q", response)
	}
}

func TestClient_Chat_ErrorResponse(t *testing.T) {
	// Create a mock HTTP server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := ChatCompletionResponse{
			Error: &struct {
				Message string `json:"message"`
				Type    string `json:"type"`
			}{
				Message: "Test error",
				Type:    "invalid_request",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-model", 0.7)
	messages := []Message{
		{Role: "user", Content: "Hello"},
	}

	ctx := context.Background()
	_, err := client.Chat(ctx, messages)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestGetSystemPrompt(t *testing.T) {
	prompt := GetSystemPrompt()
	if prompt == "" {
		t.Error("Expected non-empty system prompt")
	}
	if len(prompt) < 50 {
		t.Errorf("Expected system prompt to be substantial, got length %d", len(prompt))
	}
}
