package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
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

	// Create command
	s.cmd = exec.Command(s.serverPath, s.args...)

	// Redirect output based on debug mode
	if s.debug {
		// In debug mode, redirect to stderr so we can see it
		s.cmd.Stdout = os.Stderr
		s.cmd.Stderr = os.Stderr
	} else {
		// In normal mode, redirect to /dev/null to hide output
		devNull, err := os.OpenFile("/dev/null", os.O_WRONLY, 0)
		if err == nil {
			s.cmd.Stdout = devNull
			s.cmd.Stderr = devNull
		} else {
			// Fallback if /dev/null doesn't exist (shouldn't happen on Unix)
			s.cmd.Stdout = nil
			s.cmd.Stderr = nil
		}
	}

	// Start the process
	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start llama-server: %w", err)
	}

	fmt.Fprintf(os.Stderr, "%sWaiting for server to be ready...%s\n", colorYellow, colorReset)

	// Wait for server to be ready
	if err := s.waitForReady(30 * time.Second); err != nil {
		s.Stop() // Try to stop if we failed to start
		return fmt.Errorf("server failed to become ready: %w", err)
	}

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

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for server to be ready")
		case <-ticker.C:
			if s.isRunning() {
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

// CheckRunning checks if a server is already running at the given URL
func CheckRunning(baseURL string) bool {
	s := &Server{baseURL: baseURL}
	return s.isRunning()
}
