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

// ANSI color codes for highlighting (additional to those in server.go)
const (
	colorCyan    = "\033[36m"
	colorMagenta = "\033[35m"
	colorReverse = "\033[7m" // Reverse video for selection (may cause issues, using background instead)
)

// interactiveSelect displays a simple menu and allows selection using numbers
func interactiveSelect(title string, items []string, descriptions []string, defaultIndex int) (int, error) {
	if len(items) == 0 {
		return 0, fmt.Errorf("no items to select from")
	}
	if defaultIndex < 0 || defaultIndex >= len(items) {
		defaultIndex = 0
	}

	// Always use simple text-based selection - no fancy terminal stuff
	return simpleSelect(title, items, descriptions, defaultIndex)
}

// simpleSelect displays a simple numbered menu for selection
func simpleSelect(title string, items []string, descriptions []string, defaultIndex int) (int, error) {
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "════════════════════════════════════════════════════════════\n")
	fmt.Fprintf(os.Stderr, "  %s\n", title)
	fmt.Fprintf(os.Stderr, "════════════════════════════════════════════════════════════\n")
	fmt.Fprintf(os.Stderr, "\n")

	// Print menu items with numbers
	for i, item := range items {
		defaultMark := ""
		if i == defaultIndex {
			defaultMark = " [default]"
		}
		fmt.Fprintf(os.Stderr, "  %d) %s%s\n", i+1, item, defaultMark)
		if i < len(descriptions) && descriptions[i] != "" {
			fmt.Fprintf(os.Stderr, "     %s\n", descriptions[i])
		}
		fmt.Fprintf(os.Stderr, "\n")
	}

	// Prompt for input
	fmt.Fprintf(os.Stderr, "Enter your choice (1-%d", len(items))
	if defaultIndex >= 0 {
		fmt.Fprintf(os.Stderr, ", or Enter for %d", defaultIndex+1)
	}
	fmt.Fprintf(os.Stderr, "): ")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return defaultIndex, fmt.Errorf("failed to read input")
	}

	choice := strings.TrimSpace(scanner.Text())
	if choice == "" {
		return defaultIndex, nil
	}

	var index int
	if _, err := fmt.Sscanf(choice, "%d", &index); err != nil || index < 1 || index > len(items) {
		return defaultIndex, fmt.Errorf("invalid choice: %s (please choose 1-%d)", choice, len(items))
	}

	return index - 1, nil
}

// SelectModel interactively prompts the user to select a model in two steps:
// 1. First select the model family (using arrow keys or numbers)
// 2. Then select the size variant (using arrow keys or numbers)
func SelectModel() (string, error) {
	// Step 1: Select model family
	familyItems := make([]string, len(AvailableModelFamilies))
	familyDescriptions := make([]string, len(AvailableModelFamilies))
	for i, family := range AvailableModelFamilies {
		familyItems[i] = family.Name
		familyDescriptions[i] = family.Description
	}

	familyIndex, err := interactiveSelect("No LLM server detected. Please select a model family:", familyItems, familyDescriptions, 0)
	if err != nil {
		return "", err
	}

	selectedFamily := AvailableModelFamilies[familyIndex]

	// Step 2: Select model size
	sizeItems := make([]string, len(selectedFamily.Sizes))
	sizeDescriptions := make([]string, len(selectedFamily.Sizes))
	for i, size := range selectedFamily.Sizes {
		sizeItems[i] = fmt.Sprintf("%s (%s, %s)", size.Name, size.Description, size.Size)
		sizeDescriptions[i] = ""
	}

	// Determine default size index based on family
	defaultSizeIndex := 0
	if familyIndex == 0 {
		// For Qwen2.5-Coder, default to 3B (index 1)
		defaultSizeIndex = 1
	}

	sizeIndex, err := interactiveSelect(fmt.Sprintf("Selected: %s - Please select a size:", selectedFamily.Name), sizeItems, sizeDescriptions, defaultSizeIndex)
	if err != nil {
		return "", err
	}

	selectedSize := selectedFamily.Sizes[sizeIndex]

	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "%sSelected: %s %s%s\n", colorGreen+colorBold, selectedFamily.Name, selectedSize.Name, colorReset)
	fmt.Fprintf(os.Stderr, "\n")

	return selectedSize.ModelID, nil
}
