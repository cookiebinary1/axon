package server

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Available models for selection
var AvailableModels = map[string]string{
	"1": "Qwen/Qwen2.5-Coder-3B-Instruct-GGUF:Q4_K_M",
	"2": "bartowski/Qwen2.5-Coder-7B-Instruct-GGUF:Q4_K_M",
	"3": "Qwen/Qwen2.5-Coder-1.5B-Instruct-GGUF:Q4_K_M",
	"4": "bartowski/Qwen2.5-Coder-32B-Instruct-GGUF:Q4_K_M",
}

// SelectModel interactively prompts the user to select a model
func SelectModel() (string, error) {
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "════════════════════════════════════════════════════════════\n")
	fmt.Fprintf(os.Stderr, "  No LLM server detected. Please select a model to start:\n")
	fmt.Fprintf(os.Stderr, "════════════════════════════════════════════════════════════\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "  1) Qwen2.5-Coder-3B-Instruct (Smaller, faster, ~2GB) [default]\n")
	fmt.Fprintf(os.Stderr, "  2) Qwen2.5-Coder-7B-Instruct (Larger, better quality, ~4GB)\n")
	fmt.Fprintf(os.Stderr, "  3) Qwen2.5-Coder-1.5B-Instruct (Smallest, fastest, ~1GB)\n")
	fmt.Fprintf(os.Stderr, "  4) Qwen2.5-Coder-32B-Instruct (Largest, best quality, ~18GB)\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Enter your choice (1-4, or Enter for default): ")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return "", fmt.Errorf("failed to read input")
	}

	choice := strings.TrimSpace(scanner.Text())

	// Empty input means default (1)
	if choice == "" {
		choice = "1"
	}

	model, ok := AvailableModels[choice]
	if !ok {
		return "", fmt.Errorf("invalid choice: %s (please choose 1-4, or press Enter for default)", choice)
	}

	return model, nil
}
