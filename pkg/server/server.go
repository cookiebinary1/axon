package server

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorBlue   = "\033[34m"
	colorBold   = "\033[1m"
)

// Server represents a llama-server process
type Server struct {
	cmd        *exec.Cmd
	baseURL    string
	serverPath string
	args       []string
	debug      bool
}

// NewServer creates a new server instance
// serverPath can be empty to use "llama-server" from PATH
// debug controls whether to show server output
func NewServer(serverPath, baseURL string, model string, debug bool) *Server {
	if serverPath == "" {
		serverPath = "llama-server"
	}

	// Add --jinja flag to support tools/function calling
	args := []string{"-hf", model, "--jinja"}

	return &Server{
		serverPath: serverPath,
		baseURL:    baseURL,
		args:       args,
		debug:      debug,
	}
}

// Start starts the llama-server in the background
func (s *Server) Start() error {
	// Check if server is already running
	if s.isRunning() {
		fmt.Fprintf(os.Stderr, "%sLLM server appears to be already running at %s%s\n", colorBlue, s.baseURL, colorReset)
		return nil
	}

	fmt.Fprintf(os.Stderr, "%sStarting llama-server...%s\n", colorBlue+colorBold, colorReset)
	fmt.Fprintf(os.Stderr, "%sNote: If this is the first time using this model, it will be downloaded.%s\n", colorYellow, colorReset)
	fmt.Fprintf(os.Stderr, "%sThis may take several minutes depending on model size and internet speed.%s\n", colorYellow, colorReset)
	fmt.Fprintf(os.Stderr, "\n")

	// Create command
	s.cmd = exec.Command(s.serverPath, s.args...)

	// Create pipes to capture output
	stdoutPipe, err := s.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderrPipe, err := s.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the process
	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start llama-server: %w", err)
	}

	fmt.Fprintf(os.Stderr, "%sWaiting for server to be ready...%s\n", colorYellow, colorReset)
	fmt.Fprintf(os.Stderr, "%s(This may take a while if downloading the model)%s\n\n", colorYellow, colorReset)

	// Start goroutines to handle output
	outputDone := make(chan bool, 2)
	
	// Handle stdout - show important messages, filter out debug noise
	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			line := scanner.Text()
			// Only show lines that are likely important (download progress, errors, etc.)
			// Filter out verbose debug output like "slot update_slots", "slot launch_slot_", etc.
			if s.shouldShowOutput(line) {
				fmt.Fprintf(os.Stderr, "%s\n", line)
			}
		}
		outputDone <- true
	}()

	// Handle stderr - show important messages
	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			line := scanner.Text()
			if s.shouldShowOutput(line) {
				fmt.Fprintf(os.Stderr, "%s\n", line)
			}
		}
		outputDone <- true
	}()

	// Wait for server to be ready
	// Use longer timeout (5 minutes) to allow for model downloads
	if err := s.waitForReady(5 * time.Minute); err != nil {
		s.Stop() // Try to stop if we failed to start
		return fmt.Errorf("server failed to become ready: %w", err)
	}

	// After server is ready, continue reading output but filter it more aggressively
	// (in background, don't block)
	go func() {
		// Wait for initial output to finish
		<-outputDone
		<-outputDone
		
		// Continue reading but discard verbose output
		io.Copy(io.Discard, stdoutPipe)
		io.Copy(io.Discard, stderrPipe)
	}()

	fmt.Fprintf(os.Stderr, "%sLLM server is ready at %s%s\n", colorGreen+colorBold, s.baseURL, colorReset)
	return nil
}

// Stop stops the llama-server process
func (s *Server) Stop() error {
	if s.cmd == nil || s.cmd.Process == nil {
		return nil
	}

	fmt.Fprintf(os.Stderr, "\n%sStopping llama-server...%s\n", colorYellow+colorBold, colorReset)

	// Send SIGTERM for graceful shutdown
	if err := s.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		// If SIGTERM fails, try SIGKILL
		if err := s.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to stop server: %w", err)
		}
	}

	// Wait for process to exit (with timeout)
	done := make(chan error, 1)
	go func() {
		done <- s.cmd.Wait()
	}()

	select {
	case <-done:
		fmt.Fprintf(os.Stderr, "%sllama-server stopped%s\n", colorGreen+colorBold, colorReset)
		return nil
	case <-time.After(5 * time.Second):
		// Force kill if it doesn't stop gracefully
		if err := s.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to force stop server: %w", err)
		}
		fmt.Fprintf(os.Stderr, "%sllama-server force stopped%s\n", colorGreen+colorBold, colorReset)
		return nil
	}
}

// waitForReady waits for the server to become ready by checking the health endpoint
func (s *Server) waitForReady(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Check every 2 seconds
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	startTime := time.Now()
	lastStatusTime := startTime

	for {
		select {
		case <-ctx.Done():
			elapsed := time.Since(startTime)
			fmt.Fprintf(os.Stderr, "\n")
			return fmt.Errorf("timeout waiting for server to be ready (waited %v)", elapsed.Round(time.Second))
		case <-ticker.C:
			// Show status message every 10 seconds
			elapsed := time.Since(lastStatusTime)
			if elapsed >= 10*time.Second {
				totalElapsed := time.Since(startTime)
				fmt.Fprintf(os.Stderr, "%s[Still waiting... %v elapsed]%s\n", colorYellow, totalElapsed.Round(time.Second), colorReset)
				lastStatusTime = time.Now()
			}

			if s.isRunning() {
				fmt.Fprintf(os.Stderr, "\n")
				return nil
			}
		}
	}
}

// isRunning checks if the server is responding
// We check the /v1/models endpoint which should be available on llama-server
func (s *Server) isRunning() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Try /v1/models endpoint (OpenAI-compatible)
	req, err := http.NewRequestWithContext(ctx, "GET", s.baseURL+"/v1/models", nil)
	if err != nil {
		return false
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Accept 200 OK or any non-500 status as "running"
	return resp.StatusCode < 500
}

// SetupSignalHandling sets up signal handlers to stop the server on exit
func (s *Server) SetupSignalHandling() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		s.Stop()
		os.Exit(0)
	}()
}

// shouldShowOutput determines if a line of output should be shown to the user
// Filters out verbose debug output from llama-server
func (s *Server) shouldShowOutput(line string) bool {
	// In debug mode, show everything
	if s.debug {
		return true
	}

	// Filter out verbose slot/debug messages
	lowerLine := strings.ToLower(line)
	skipPatterns := []string{
		"slot update_slots",
		"slot launch_slot_",
		"slot get_availabl",
		"params_from_",
		"chat format:",
		"sampler chain:",
		"processing task",
		"prompt processing progress",
		"n_tokens =",
		"memory_seq_rm",
		"batch.n_tokens",
	}

	for _, pattern := range skipPatterns {
		if strings.Contains(lowerLine, pattern) {
			return false
		}
	}

	// Show everything else (errors, important messages, download progress, etc.)
	return true
}

// CheckRunning checks if a server is already running at the given URL
func CheckRunning(baseURL string) bool {
	s := &Server{baseURL: baseURL}
	return s.isRunning()
}
