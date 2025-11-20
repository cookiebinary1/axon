package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	logFile      *os.File
	logFileMutex sync.Mutex
	logEnabled   bool
)

// InitLogger initializes the debug logger if enabled
func InitLogger(projectRoot string) error {
	enabled := os.Getenv("AXON_DEBUG_LOG") == "1" || os.Getenv("AXON_DEBUG_LOG") == "true"
	if !enabled {
		return nil
	}

	logEnabled = true
	logPath := filepath.Join(projectRoot, ".axon-debug.log")

	var err error
	logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	Logf("=== AXON Debug Log Started at %s ===\n", time.Now().Format(time.RFC3339))
	return nil
}

// Logf writes a formatted log message to the debug log file
func Logf(format string, args ...interface{}) {
	if !logEnabled || logFile == nil {
		return
	}

	logFileMutex.Lock()
	defer logFileMutex.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	message := fmt.Sprintf(format, args...)
	fmt.Fprintf(logFile, "[%s] %s", timestamp, message)
	logFile.Sync() // Flush immediately
}

// LogRequest logs an HTTP request
func LogRequest(method, url string, body []byte) {
	if !logEnabled {
		return
	}

	Logf(">>> REQUEST: %s %s\n", method, url)
	if len(body) > 0 {
		// Truncate very long bodies
		bodyStr := string(body)
		if len(bodyStr) > 10000 {
			bodyStr = bodyStr[:10000] + "\n... (truncated)"
		}
		Logf(">>> BODY:\n%s\n", bodyStr)
	}
}

// LogResponse logs an HTTP response
func LogResponse(statusCode int, body string) {
	if !logEnabled {
		return
	}

	Logf("<<< RESPONSE: Status %d\n", statusCode)
	if len(body) > 0 {
		// Truncate very long bodies
		if len(body) > 10000 {
			body = body[:10000] + "\n... (truncated)"
		}
		Logf("<<< BODY:\n%s\n", body)
	}
}

// LogToolCall logs a tool call
func LogToolCall(name string, args string, result string, err error) {
	if !logEnabled {
		return
	}

	Logf("🔧 TOOL CALL: %s\n", name)
	Logf("   Args: %s\n", args)
	if err != nil {
		Logf("   Error: %v\n", err)
	} else {
		// Truncate long results
		if len(result) > 5000 {
			result = result[:5000] + "\n... (truncated)"
		}
		Logf("   Result: %s\n", result)
	}
}

// CloseLogger closes the debug log file
func CloseLogger() {
	if logFile != nil {
		logFileMutex.Lock()
		defer logFileMutex.Unlock()

		Logf("=== AXON Debug Log Ended ===\n\n")
		logFile.Close()
		logFile = nil
	}
}

// TruncateString truncates a string to maxLen characters
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "... (truncated)"
}
