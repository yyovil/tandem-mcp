package toolhandler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	dockerClient     *client.Client
	dockerClientOnce sync.Once
	dockerClientErr  error

	// Attach session management - singleton per container
	attachSession     *AttachSession
	attachSessionLock sync.Mutex
)

type AttachSession struct {
	containerID  string
	attachResp   types.HijackedResponse
	cli          *client.Client
	outputBuffer *SafeBuffer
	outputStream chan string
	streamCtx    context.Context
	streamCancel context.CancelFunc
}

// SafeBuffer is a thread-safe buffer for collecting output
type SafeBuffer struct {
	buf bytes.Buffer
	mu  sync.Mutex
}

func (sb *SafeBuffer) Write(p []byte) (n int, err error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.Write(p)
}

func (sb *SafeBuffer) String() string {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.String()
}

func (sb *SafeBuffer) Reset() {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	sb.buf.Reset()
}

const CONTAINER_IMAGE = "kali:headless"

// returns a singleton Docker client instance
func getDockerClient() (*client.Client, error) {
	dockerClientOnce.Do(func() {
		dockerClient, dockerClientErr = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	})
	return dockerClient, dockerClientErr
}

// getOrCreateAttachSession returns a persistent attach session to the container's shell
func getOrCreateAttachSession(ctx context.Context, cli *client.Client, containerID string) (*AttachSession, error) {
	attachSessionLock.Lock()
	defer attachSessionLock.Unlock()

	// Check if we already have a valid attach session for this container
	if attachSession != nil && attachSession.containerID == containerID {
		// Verify container is still running
		info, err := cli.ContainerInspect(ctx, containerID)
		if err == nil && info.State.Running {
			return attachSession, nil
		}
		// Container stopped or error, clean up
		if attachSession.streamCancel != nil {
			attachSession.streamCancel()
		}
		attachSession.attachResp.Close()
		attachSession = nil
	}

	// Attach to the running container's stdin/stdout/stderr
	// This connects to the main bash process running in the container
	attachResp, err := cli.ContainerAttach(ctx, containerID, container.AttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to attach to container: %w", err)
	}

	// Create context for streaming goroutine
	streamCtx, streamCancel := context.WithCancel(context.Background())

	outputBuffer := &SafeBuffer{}
	outputStream := make(chan string, 100)

	session := &AttachSession{
		containerID:  containerID,
		attachResp:   attachResp,
		cli:          cli,
		outputBuffer: outputBuffer,
		outputStream: outputStream,
		streamCtx:    streamCtx,
		streamCancel: streamCancel,
	}

	// Start output streaming goroutine (like Docker CLI does)
	go session.beginOutputStream()

	// Give bash time to start and drain initial prompt
	time.Sleep(200 * time.Millisecond)
	outputBuffer.Reset() // Clear any initial output

	attachSession = session
	log.Printf("Created attach session with streaming to container: %s", containerID)
	return attachSession, nil
}

// beginOutputStream continuously reads from the container and writes to buffer
// This mimics Docker CLI's approach
func (s *AttachSession) beginOutputStream() {
	defer close(s.outputStream)

	buf := make([]byte, 4096)
	for {
		select {
		case <-s.streamCtx.Done():
			return
		default:
			// Read from container output (TTY mode - raw stream)
			n, err := s.attachResp.Reader.Read(buf)
			if n > 0 {
				// Write to buffer
				s.outputBuffer.Write(buf[:n])
			}
			if err != nil {
				if err != io.EOF {
					log.Printf("Output stream error: %v", err)
				}
				return
			}
		}
	}
}

type TerminalArgs struct {
	Command  string   `json:"command" jsonschema:"The command to execute,required"`
	Argument []string `json:"argument,omitempty" jsonschema:"Arguments for the command"`
}

func Terminal(ctx context.Context, request *mcp.CallToolRequest, args TerminalArgs) (result *mcp.CallToolResult, output any, err error) {
	result = &mcp.CallToolResult{}

	cli, err := getDockerClient()
	if err != nil {
		result.Content = []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Failed to create Docker client: %v", err)},
		}
		result.IsError = true
		return
	}

	var containerID string

	containerSummaries, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		result.Content = []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Failed to list containers: %v", err)},
		}
		result.IsError = true
		return
	}

	var existingContainer *container.Summary
	for _, containerSummary := range containerSummaries {
		if containerSummary.Image == CONTAINER_IMAGE {
			existingContainer = &containerSummary
			break
		}
	}

	if existingContainer != nil {
		containerID = existingContainer.ID
		state := existingContainer.State

		switch state {
		case "created":
			log.Printf("Starting created container: %s", containerID)
			if err = cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
				result.Content = []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Failed to start created container: %v", err)},
				}
				result.IsError = true
				return
			}
		case "paused":
			log.Printf("Unpausing paused container: %s", containerID)
			if err = cli.ContainerUnpause(ctx, containerID); err != nil {
				result.Content = []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Failed to unpause container: %v", err)},
				}
				result.IsError = true
				return
			}
		case "exited":
			log.Printf("Restarting exited container: %s", containerID)
			if err = cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
				result.Content = []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Failed to restart exited container: %v", err)},
				}
				result.IsError = true
				return
			}
		}
	} else {
		log.Printf("Creating new Kali container")
		var resp container.CreateResponse
		resp, err = cli.ContainerCreate(ctx, &container.Config{
			Image:      CONTAINER_IMAGE,
			Cmd:        []string{"bash"},
			Tty:        true,       // TTY for raw terminal output
			OpenStdin:  true,       // Keep stdin open
			StdinOnce:  false,      // Don't close stdin after first attach
			Entrypoint: []string{}, // Use default entrypoint
		}, nil, nil, nil, "")
		if err != nil {
			result.Content = []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Failed to create container: %v", err)},
			}
			result.IsError = true
			return
		}

		containerID = resp.ID

		if err = cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
			result.Content = []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Failed to start container: %v", err)},
			}
			result.IsError = true
			return
		}
		log.Printf("Created and started new container: %s", containerID)
	}

	// Get or create attach session to the container's bash shell
	session, err := getOrCreateAttachSession(ctx, cli, containerID)
	if err != nil {
		result.Content = []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Failed to get attach session: %v", err)},
		}
		result.IsError = true
		return
	}

	// Build the command string
	cmdString := args.Command
	if len(args.Argument) > 0 {
		cmdString = args.Command + " " + strings.Join(args.Argument, " ")
	}

	// Execute command and capture output
	outputStr, err := session.executeCommand(cmdString)
	if err != nil {
		result.Content = []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Failed to execute command: %v", err)},
		}
		result.IsError = true
		return
	}

	result.Content = []mcp.Content{
		&mcp.TextContent{Text: outputStr},
	}

	return
}

// executeCommand sends a command and waits for completion by detecting the prompt
func (s *AttachSession) executeCommand(command string) (string, error) {
	// Clear the output buffer before sending command
	s.outputBuffer.Reset()

	// Send command with newline (like Docker CLI sends input)
	_, err := io.WriteString(s.attachResp.Conn, command+"\n")
	if err != nil {
		return "", fmt.Errorf("failed to write command: %w", err)
	}

	// Wait for command to execute and prompt to reappear
	// Use a timeout-based approach since we can't detect prompt reliably
	time.Sleep(500 * time.Millisecond)

	// Start collecting output
	startTime := time.Now()
	lastOutputTime := startTime
	maxWaitTime := 30 * time.Second
	idleTimeout := 500 * time.Millisecond
	lastLen := 0

	for {
		currentOutput := s.outputBuffer.String()
		currentLen := len(currentOutput)

		// Check if we have new output
		if currentLen > lastLen {
			lastOutputTime = time.Now()
			lastLen = currentLen
		}

		// Check if we've been idle (no new output) for the idle timeout
		if time.Since(lastOutputTime) > idleTimeout {
			// Command likely finished
			break
		}

		// Check overall timeout
		if time.Since(startTime) > maxWaitTime {
			break
		}

		// Small sleep to avoid busy waiting
		time.Sleep(50 * time.Millisecond)
	}

	// Get final output
	rawOutput := s.outputBuffer.String()

	// Clean up output (remove command echo, ANSI codes, and prompts)
	cleanedOutput := cleanOutput(rawOutput, command)

	return cleanedOutput, nil
}

// cleanOutput removes the command echo, ANSI codes, and bash prompts from the output
func cleanOutput(output, command string) string {
	// Remove ANSI escape sequences (colors, cursor movements, bracketed paste mode, etc.)
	// Comprehensive pattern for all ESC sequences
	ansiRegex := regexp.MustCompile(`\x1b\[[^a-zA-Z]*[a-zA-Z]|\x1b\][^\a]*\a`)
	output = ansiRegex.ReplaceAllString(output, "")

	// Split by newlines (handle both \r\n and \n)
	output = strings.ReplaceAll(output, "\r\n", "\n")
	output = strings.ReplaceAll(output, "\r", "\n")

	lines := strings.Split(output, "\n")
	var cleanedLines []string
	seenCommand := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines
		if trimmed == "" {
			continue
		}

		// Skip the line that echoes the command (first occurrence)
		if !seenCommand && strings.Contains(line, command) {
			seenCommand = true
			continue
		}

		// Skip bash prompts (including fancy Kali prompts)
		if strings.HasPrefix(trimmed, "┌") || // Kali top line
			strings.HasPrefix(trimmed, "└") || // Kali bottom line
			strings.Contains(trimmed, "㉿") || // Kali specific character
			strings.HasPrefix(trimmed, "bash-") ||
			strings.HasPrefix(trimmed, "root@") ||
			strings.HasSuffix(trimmed, "$") || // Ends with $
			strings.HasSuffix(trimmed, "#") { // Ends with #
			continue
		}

		cleanedLines = append(cleanedLines, trimmed)
	}

	return strings.TrimSpace(strings.Join(cleanedLines, "\n"))
}
