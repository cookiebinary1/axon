package server

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
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
	colorReverse = "\033[7m" // Reverse video for selection
)

// interactiveSelect displays an interactive menu and allows selection using arrow keys or numbers
func interactiveSelect(title string, items []string, descriptions []string, defaultIndex int) (int, error) {
	if len(items) == 0 {
		return 0, fmt.Errorf("no items to select from")
	}
	if defaultIndex < 0 || defaultIndex >= len(items) {
		defaultIndex = 0
	}

	// Check if stdin is a terminal
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		// Fallback to simple number input if not a terminal
		return simpleSelect(title, items, descriptions, defaultIndex)
	}

	// Save original terminal state
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		// Fallback to simple input if raw mode fails
		return simpleSelect(title, items, descriptions, defaultIndex)
	}

	// Ensure terminal is restored even on panic
	defer func() {
		term.Restore(fd, oldState)
		// Show cursor again
		fmt.Fprintf(os.Stderr, "\033[?25h")
	}()

	// Hide cursor for cleaner menu
	fmt.Fprintf(os.Stderr, "\033[?25l")

	selected := defaultIndex
	reader := bufio.NewReader(os.Stdin)

	// Print initial menu
	printMenu(title, items, descriptions, selected)

	for {
		// Read a single character
		char, err := reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				return selected, nil
			}
			return selected, err
		}

		// Handle escape sequences (arrow keys)
		if char == 0x1b { // ESC
			// Read the next two bytes to determine the arrow key
			// Use ReadByte to ensure we get all bytes
			next1, err1 := reader.ReadByte()
			if err1 != nil {
				continue
			}
			if next1 != '[' {
				// Not an arrow key sequence, ignore
				continue
			}
			next2, err2 := reader.ReadByte()
			if err2 != nil {
				continue
			}
			switch next2 {
			case 'A': // Up arrow
				if selected > 0 {
					selected--
				} else {
					selected = len(items) - 1
				}
				redrawMenu(title, items, descriptions, selected)
			case 'B': // Down arrow
				if selected < len(items)-1 {
					selected++
				} else {
					selected = 0
				}
				redrawMenu(title, items, descriptions, selected)
			}
			continue
		}

		// Handle Enter key
		if char == '\r' || char == '\n' {
			// Clear the menu before returning
			fmt.Fprintf(os.Stderr, "\033[2J\033[H\033[?25h")
			return selected, nil
		}

		// Handle number keys (1-9) - immediately confirm selection
		if char >= '1' && char <= '9' {
			num := int(char - '0')
			if num <= len(items) {
				selected = num - 1
				// Clear the menu before returning
				fmt.Fprintf(os.Stderr, "\033[2J\033[H\033[?25h")
				return selected, nil
			}
		}

		// Handle Ctrl+C
		if char == 0x03 {
			return selected, fmt.Errorf("interrupted by user")
		}
	}
}

// simpleSelect is a fallback for non-terminal input
func simpleSelect(title string, items []string, descriptions []string, defaultIndex int) (int, error) {
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "════════════════════════════════════════════════════════════\n")
	fmt.Fprintf(os.Stderr, "  %s\n", title)
	fmt.Fprintf(os.Stderr, "════════════════════════════════════════════════════════════\n")
	fmt.Fprintf(os.Stderr, "\n")

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

	fmt.Fprintf(os.Stderr, "Enter your choice (1-%d, or Enter for default): ", len(items))

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
		return defaultIndex, fmt.Errorf("invalid choice: %s", choice)
	}

	return index - 1, nil
}

// printMenu prints the menu with the selected item highlighted
func printMenu(title string, items []string, descriptions []string, selected int) {
	// Only clear screen if this is the first print (not a redraw)
	// We'll use a different approach - save cursor position and restore
	fmt.Fprintf(os.Stderr, "\033[2J\033[H") // Clear screen and move to top
	fmt.Fprintf(os.Stderr, "════════════════════════════════════════════════════════════\n")
	fmt.Fprintf(os.Stderr, "  %s\n", title)
	fmt.Fprintf(os.Stderr, "════════════════════════════════════════════════════════════\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "%sUse ↑↓ arrows or numbers to select, Enter to confirm%s\n", colorYellow, colorReset)
	fmt.Fprintf(os.Stderr, "\n")

	for i, item := range items {
		if i == selected {
			fmt.Fprintf(os.Stderr, "%s%s  ▶ %s%s\n", colorReverse+colorBold, item, colorReset, colorReset)
		} else {
			fmt.Fprintf(os.Stderr, "     %s\n", item)
		}
		if i < len(descriptions) && descriptions[i] != "" {
			if i == selected {
				fmt.Fprintf(os.Stderr, "%s     %s%s\n", colorReverse, descriptions[i], colorReset)
			} else {
				fmt.Fprintf(os.Stderr, "     %s\n", descriptions[i])
			}
		}
		fmt.Fprintf(os.Stderr, "\n")
	}
}

// redrawMenu updates the menu display with new selection
func redrawMenu(title string, items []string, descriptions []string, selected int) {
	// Calculate how many lines we need to clear
	// Header: 3 lines (separator, title, separator)
	// Instructions: 2 lines (text + blank)
	// Items: each item takes 2 lines (item + blank), or 3 if it has description
	lines := 3 + 2 // header + instructions
	for i := 0; i < len(items); i++ {
		lines++ // item line
		if i < len(descriptions) && descriptions[i] != "" {
			lines++ // description line
		}
		lines++ // blank line after each item
	}

	// Move cursor up and clear
	fmt.Fprintf(os.Stderr, "\033[%dA", lines)
	fmt.Fprintf(os.Stderr, "\033[J")

	// Re-print menu
	printMenu(title, items, descriptions, selected)
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
