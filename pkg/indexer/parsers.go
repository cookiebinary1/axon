package indexer

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

// Go parser
func parseGoFile(path string) []Symbol {
	symbols := []Symbol{}
	file, err := os.Open(path)
	if err != nil {
		return symbols
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	// Patterns for Go
	funcPattern := regexp.MustCompile(`^\s*func\s+(\([^)]+\)\s+)?([A-Za-z_][A-Za-z0-9_]*)`)
	typePattern := regexp.MustCompile(`^\s*type\s+([A-Za-z_][A-Za-z0-9_]*)\s+(struct|interface)`)
	
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Match functions
		if matches := funcPattern.FindStringSubmatch(line); matches != nil {
			funcName := matches[len(matches)-1]
			symbols = append(symbols, Symbol{
				Name: funcName,
				Type: "function",
				Line: lineNum,
			})
		}

		// Match types (structs/interfaces)
		if matches := typePattern.FindStringSubmatch(line); matches != nil {
			typeName := matches[1]
			typeType := matches[2]
			symbols = append(symbols, Symbol{
				Name: typeName,
				Type: typeType,
				Line: lineNum,
			})
		}
	}

	return symbols
}

// PHP parser
func parsePHPFile(path string) []Symbol {
	symbols := []Symbol{}
	file, err := os.Open(path)
	if err != nil {
		return symbols
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	// Patterns for PHP
	classPattern := regexp.MustCompile(`^\s*(?:abstract\s+|final\s+)?class\s+([A-Za-z_][A-Za-z0-9_]*)`)
	funcPattern := regexp.MustCompile(`^\s*(?:public|private|protected|static|\s)*\s*function\s+([A-Za-z_][A-Za-z0-9_]*)`)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Match classes
		if matches := classPattern.FindStringSubmatch(line); matches != nil {
			symbols = append(symbols, Symbol{
				Name: matches[1],
				Type: "class",
				Line: lineNum,
			})
		}

		// Match functions/methods
		if matches := funcPattern.FindStringSubmatch(line); matches != nil {
			symbols = append(symbols, Symbol{
				Name: matches[1],
				Type: "function",
				Line: lineNum,
			})
		}
	}

	return symbols
}

// JavaScript parser
func parseJavaScriptFile(path string) []Symbol {
	symbols := []Symbol{}
	file, err := os.Open(path)
	if err != nil {
		return symbols
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	// Patterns for JavaScript
	funcPattern := regexp.MustCompile(`^\s*(?:export\s+)?(?:async\s+)?function\s+([A-Za-z_$][A-Za-z0-9_$]*)`)
	arrowFuncPattern := regexp.MustCompile(`^\s*(?:export\s+)?(?:const|let|var)\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*=\s*(?:async\s*)?\(`)
	classPattern := regexp.MustCompile(`^\s*(?:export\s+)?class\s+([A-Za-z_$][A-Za-z0-9_$]*)`)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Match classes
		if matches := classPattern.FindStringSubmatch(line); matches != nil {
			symbols = append(symbols, Symbol{
				Name: matches[1],
				Type: "class",
				Line: lineNum,
			})
		}

		// Match regular functions
		if matches := funcPattern.FindStringSubmatch(line); matches != nil {
			symbols = append(symbols, Symbol{
				Name: matches[1],
				Type: "function",
				Line: lineNum,
			})
		}

		// Match arrow functions
		if matches := arrowFuncPattern.FindStringSubmatch(line); matches != nil {
			symbols = append(symbols, Symbol{
				Name: matches[1],
				Type: "function",
				Line: lineNum,
			})
		}
	}

	return symbols
}

// TypeScript parser (similar to JS but with type annotations)
func parseTypeScriptFile(path string) []Symbol {
	return parseJavaScriptFile(path) // Similar patterns, reuse JS parser
}

// C/C++ parser
func parseCFile(path string) []Symbol {
	symbols := []Symbol{}
	file, err := os.Open(path)
	if err != nil {
		return symbols
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	// Patterns for C/C++
	funcPattern := regexp.MustCompile(`^\s*(?:static\s+|inline\s+)?(?:[a-zA-Z_][a-zA-Z0-9_]*\s+)+([a-zA-Z_][a-zA-Z0-9_]*)\s*\(`)
	structPattern := regexp.MustCompile(`^\s*struct\s+([a-zA-Z_][a-zA-Z0-9_]*)`)
	classPattern := regexp.MustCompile(`^\s*class\s+([a-zA-Z_][a-zA-Z0-9_]*)`)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Match functions
		if matches := funcPattern.FindStringSubmatch(line); matches != nil {
			symbols = append(symbols, Symbol{
				Name: matches[1],
				Type: "function",
				Line: lineNum,
			})
		}

		// Match structs
		if matches := structPattern.FindStringSubmatch(line); matches != nil {
			symbols = append(symbols, Symbol{
				Name: matches[1],
				Type: "struct",
				Line: lineNum,
			})
		}

		// Match classes (C++)
		if matches := classPattern.FindStringSubmatch(line); matches != nil {
			symbols = append(symbols, Symbol{
				Name: matches[1],
				Type: "class",
				Line: lineNum,
			})
		}
	}

	return symbols
}

// Lua parser
func parseLuaFile(path string) []Symbol {
	symbols := []Symbol{}
	file, err := os.Open(path)
	if err != nil {
		return symbols
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	// Patterns for Lua
	funcPattern := regexp.MustCompile(`^\s*(?:local\s+)?function\s+([A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*)`)
	assignmentPattern := regexp.MustCompile(`^\s*(?:local\s+)?([A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*)\s*=\s*function\s*\(`)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Match functions
		if matches := funcPattern.FindStringSubmatch(line); matches != nil {
			symbols = append(symbols, Symbol{
				Name: matches[1],
				Type: "function",
				Line: lineNum,
			})
		}

		// Match function assignments
		if matches := assignmentPattern.FindStringSubmatch(line); matches != nil {
			symbols = append(symbols, Symbol{
				Name: matches[1],
				Type: "function",
				Line: lineNum,
			})
		}
	}

	return symbols
}

// Python parser
func parsePythonFile(path string) []Symbol {
	symbols := []Symbol{}
	file, err := os.Open(path)
	if err != nil {
		return symbols
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	// Patterns for Python
	classPattern := regexp.MustCompile(`^\s*class\s+([A-Za-z_][A-Za-z0-9_]*)`)
	funcPattern := regexp.MustCompile(`^\s*def\s+([A-Za-z_][A-Za-z0-9_]*)`)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Match classes
		if matches := classPattern.FindStringSubmatch(line); matches != nil {
			symbols = append(symbols, Symbol{
				Name: matches[1],
				Type: "class",
				Line: lineNum,
			})
		}

		// Match functions
		if matches := funcPattern.FindStringSubmatch(line); matches != nil {
			symbols = append(symbols, Symbol{
				Name: matches[1],
				Type: "function",
				Line: lineNum,
			})
		}
	}

	return symbols
}

// Java parser
func parseJavaFile(path string) []Symbol {
	symbols := []Symbol{}
	file, err := os.Open(path)
	if err != nil {
		return symbols
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	// Patterns for Java
	classPattern := regexp.MustCompile(`^\s*(?:public|private|protected|abstract|final|\s)*\s*class\s+([A-Za-z_][A-Za-z0-9_]*)`)
	funcPattern := regexp.MustCompile(`^\s*(?:public|private|protected|static|abstract|\s)*\s*(?:[A-Za-z_][A-Za-z0-9_.]*\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*\(`)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Match classes
		if matches := classPattern.FindStringSubmatch(line); matches != nil {
			symbols = append(symbols, Symbol{
				Name: matches[1],
				Type: "class",
				Line: lineNum,
			})
		}

		// Match methods (simplified)
		if matches := funcPattern.FindStringSubmatch(line); matches != nil {
			// Skip if it's a class definition
			if !strings.Contains(line, "class") {
				symbols = append(symbols, Symbol{
					Name: matches[len(matches)-1],
					Type: "method",
					Line: lineNum,
				})
			}
		}
	}

	return symbols
}

// Ruby parser
func parseRubyFile(path string) []Symbol {
	symbols := []Symbol{}
	file, err := os.Open(path)
	if err != nil {
		return symbols
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	// Patterns for Ruby
	classPattern := regexp.MustCompile(`^\s*class\s+([A-Za-z_][A-Za-z0-9_]*(?:::[A-Za-z_][A-Za-z0-9_]*)*)`)
	funcPattern := regexp.MustCompile(`^\s*def\s+(?:self\.)?([A-Za-z_][A-Za-z0-9_]*)`)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Match classes
		if matches := classPattern.FindStringSubmatch(line); matches != nil {
			symbols = append(symbols, Symbol{
				Name: matches[1],
				Type: "class",
				Line: lineNum,
			})
		}

		// Match methods
		if matches := funcPattern.FindStringSubmatch(line); matches != nil {
			symbols = append(symbols, Symbol{
				Name: matches[1],
				Type: "function",
				Line: lineNum,
			})
		}
	}

	return symbols
}

// Rust parser
func parseRustFile(path string) []Symbol {
	symbols := []Symbol{}
	file, err := os.Open(path)
	if err != nil {
		return symbols
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	// Patterns for Rust
	structPattern := regexp.MustCompile(`^\s*(?:pub\s+)?struct\s+([A-Za-z_][A-Za-z0-9_]*)`)
	implPattern := regexp.MustCompile(`^\s*impl\s+(?:[A-Za-z_][A-Za-z0-9_]*::)?([A-Za-z_][A-Za-z0-9_]*)`)
	funcPattern := regexp.MustCompile(`^\s*(?:pub\s+)?fn\s+([A-Za-z_][A-Za-z0-9_]*)`)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Match structs
		if matches := structPattern.FindStringSubmatch(line); matches != nil {
			symbols = append(symbols, Symbol{
				Name: matches[1],
				Type: "struct",
				Line: lineNum,
			})
		}

		// Match impl blocks
		if matches := implPattern.FindStringSubmatch(line); matches != nil {
			symbols = append(symbols, Symbol{
				Name: matches[1],
				Type: "impl",
				Line: lineNum,
			})
		}

		// Match functions
		if matches := funcPattern.FindStringSubmatch(line); matches != nil {
			symbols = append(symbols, Symbol{
				Name: matches[1],
				Type: "function",
				Line: lineNum,
			})
		}
	}

	return symbols
}

// Shell parser
func parseShellFile(path string) []Symbol {
	symbols := []Symbol{}
	file, err := os.Open(path)
	if err != nil {
		return symbols
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	// Patterns for shell scripts
	funcPattern := regexp.MustCompile(`^\s*function\s+([A-Za-z_][A-Za-z0-9_]*)`)
	funcPattern2 := regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_]*)\s*\(\)\s*\{?`)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip comments
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}

		// Match functions
		if matches := funcPattern.FindStringSubmatch(line); matches != nil {
			symbols = append(symbols, Symbol{
				Name: matches[1],
				Type: "function",
				Line: lineNum,
			})
		} else if matches := funcPattern2.FindStringSubmatch(line); matches != nil {
			symbols = append(symbols, Symbol{
				Name: matches[1],
				Type: "function",
				Line: lineNum,
			})
		}
	}

	return symbols
}

