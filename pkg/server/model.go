package server

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ModelSize represents a model size variant
type ModelSize struct {
	Name        string
	ModelID     string
	Description string
	Size        string
}

// ModelFamily represents a model family with multiple size variants
type ModelFamily struct {
	Name        string
	Description string
	Sizes       []ModelSize
}

// AvailableModelFamilies contains all available model families
var AvailableModelFamilies = []ModelFamily{
	{
		Name:        "Qwen2.5-Coder",
		Description: "Alibaba's coding model, excellent for code generation and understanding",
		Sizes: []ModelSize{
			{Name: "1.5B", ModelID: "Qwen/Qwen2.5-Coder-1.5B-Instruct-GGUF:Q4_K_M", Description: "Smallest, fastest", Size: "~1GB"},
			{Name: "3B", ModelID: "Qwen/Qwen2.5-Coder-3B-Instruct-GGUF:Q4_K_M", Description: "Smaller, faster", Size: "~2GB"},
			{Name: "7B", ModelID: "bartowski/Qwen2.5-Coder-7B-Instruct-GGUF:Q4_K_M", Description: "Larger, better quality", Size: "~4GB"},
			{Name: "32B", ModelID: "bartowski/Qwen2.5-Coder-32B-Instruct-GGUF:Q4_K_M", Description: "Largest, best quality", Size: "~18GB"},
		},
	},
	{
		Name:        "DeepSeek Coder",
		Description: "DeepSeek's specialized coding model, great for complex code tasks",
		Sizes: []ModelSize{
			{Name: "1.3B", ModelID: "bartowski/DeepSeek-Coder-1.3B-Instruct-GGUF:Q4_K_M", Description: "Smallest, fastest", Size: "~1GB"},
			{Name: "6.7B", ModelID: "bartowski/DeepSeek-Coder-6.7B-Instruct-GGUF:Q4_K_M", Description: "Good balance", Size: "~4GB"},
			{Name: "33B", ModelID: "bartowski/DeepSeek-Coder-33B-Instruct-GGUF:Q4_K_M", Description: "Largest, best quality", Size: "~18GB"},
		},
	},
	{
		Name:        "CodeLlama",
		Description: "Meta's coding model based on Llama, good general-purpose coding",
		Sizes: []ModelSize{
			{Name: "7B", ModelID: "bartowski/CodeLlama-7B-Instruct-GGUF:Q4_K_M", Description: "Good balance", Size: "~4GB"},
			{Name: "13B", ModelID: "bartowski/CodeLlama-13B-Instruct-GGUF:Q4_K_M", Description: "Larger, better quality", Size: "~7GB"},
			{Name: "34B", ModelID: "bartowski/CodeLlama-34B-Instruct-GGUF:Q4_K_M", Description: "Largest, best quality", Size: "~18GB"},
		},
	},
	{
		Name:        "StarCoder",
		Description: "BigCode's StarCoder, trained on permissively licensed code",
		Sizes: []ModelSize{
			{Name: "3B", ModelID: "bartowski/starcoder2-3b-GGUF:Q4_K_M", Description: "Smaller, faster", Size: "~2GB"},
			{Name: "7B", ModelID: "bartowski/starcoder2-7b-GGUF:Q4_K_M", Description: "Good balance", Size: "~4GB"},
			{Name: "15B", ModelID: "bartowski/starcoder2-15b-GGUF:Q4_K_M", Description: "Larger, better quality", Size: "~8GB"},
		},
	},
}

// SelectModel interactively prompts the user to select a model in two steps:
// 1. First select the model family
// 2. Then select the size variant
func SelectModel() (string, error) {
	scanner := bufio.NewScanner(os.Stdin)

	// Step 1: Select model family
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "════════════════════════════════════════════════════════════\n")
	fmt.Fprintf(os.Stderr, "  No LLM server detected. Please select a model family:\n")
	fmt.Fprintf(os.Stderr, "════════════════════════════════════════════════════════════\n")
	fmt.Fprintf(os.Stderr, "\n")

	for i, family := range AvailableModelFamilies {
		defaultMark := ""
		if i == 0 {
			defaultMark = " [default]"
		}
		fmt.Fprintf(os.Stderr, "  %d) %s%s\n", i+1, family.Name, defaultMark)
		fmt.Fprintf(os.Stderr, "     %s\n", family.Description)
		fmt.Fprintf(os.Stderr, "\n")
	}

	fmt.Fprintf(os.Stderr, "Enter your choice (1-%d, or Enter for default): ", len(AvailableModelFamilies))

	if !scanner.Scan() {
		return "", fmt.Errorf("failed to read input")
	}

	familyChoice := strings.TrimSpace(scanner.Text())

	// Empty input means default (1)
	if familyChoice == "" {
		familyChoice = "1"
	}

	var familyIndex int
	if _, err := fmt.Sscanf(familyChoice, "%d", &familyIndex); err != nil || familyIndex < 1 || familyIndex > len(AvailableModelFamilies) {
		return "", fmt.Errorf("invalid choice: %s (please choose 1-%d, or press Enter for default)", familyChoice, len(AvailableModelFamilies))
	}

	selectedFamily := AvailableModelFamilies[familyIndex-1]

	// Step 2: Select model size
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "════════════════════════════════════════════════════════════\n")
	fmt.Fprintf(os.Stderr, "  Selected: %s\n", selectedFamily.Name)
	fmt.Fprintf(os.Stderr, "  Please select a size:\n")
	fmt.Fprintf(os.Stderr, "════════════════════════════════════════════════════════════\n")
	fmt.Fprintf(os.Stderr, "\n")

	// Determine default size index based on family
	defaultSizeIndex := 0
	if familyIndex == 1 {
		// For Qwen2.5-Coder, default to 3B (index 1)
		defaultSizeIndex = 1
	}

	for i, size := range selectedFamily.Sizes {
		defaultMark := ""
		if i == defaultSizeIndex {
			defaultMark = " [default]"
		}
		fmt.Fprintf(os.Stderr, "  %d) %s (%s, %s)%s\n", i+1, size.Name, size.Description, size.Size, defaultMark)
	}

	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Enter your choice (1-%d, or Enter for default): ", len(selectedFamily.Sizes))

	if !scanner.Scan() {
		return "", fmt.Errorf("failed to read input")
	}

	sizeChoice := strings.TrimSpace(scanner.Text())

	// Empty input means default
	if sizeChoice == "" {
		sizeChoice = fmt.Sprintf("%d", defaultSizeIndex+1)
	}

	var sizeIndex int
	if _, err := fmt.Sscanf(sizeChoice, "%d", &sizeIndex); err != nil || sizeIndex < 1 || sizeIndex > len(selectedFamily.Sizes) {
		return "", fmt.Errorf("invalid choice: %s (please choose 1-%d, or press Enter for default)", sizeChoice, len(selectedFamily.Sizes))
	}

	selectedSize := selectedFamily.Sizes[sizeIndex-1]

	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "%sSelected: %s %s%s\n", "\033[32m\033[1m", selectedFamily.Name, selectedSize.Name, "\033[0m")
	fmt.Fprintf(os.Stderr, "\n")

	return selectedSize.ModelID, nil
}
