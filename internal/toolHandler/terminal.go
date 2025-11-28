package toolhandler

import (
	"bytes"
	"context"
	"encoding/base64"
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

	// Attached session management (singleton per container)
	attachedSessions     = make(map[string]*AttachedSession)
	attachedSessionsLock sync.RWMutex
)

const CONTAINER_IMAGE = "kali:headless"
const TMUX_SESSION = "pentest"
const AGENT_WINDOW = "agent"
const SCREENSHOTTER_WINDOW = "screenshotter"
const PROMPT_MARKER = "KALI[" // Prompt format: KALI[path]#

// promptRegex matches the prompt pattern KALI[...]#
var promptRegex = regexp.MustCompile(`^KALI\[.*\]#\s*$`)

// AttachedSession represents a persistent docker attach connection to a container
// We use docker attach for stdin (sending commands) and tmux capture-pane for reading output
type AttachedSession struct {
	containerID  string
	hijackedResp types.HijackedResponse
	ctx          context.Context
	cancel       context.CancelFunc
}

// returns a singleton Docker client instance
func getDockerClient() (*client.Client, error) {
	dockerClientOnce.Do(func() {
		dockerClient, dockerClientErr = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	})
	return dockerClient, dockerClientErr
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

	containerID, err := ensureContainerRunning(ctx, cli)
	if err != nil {
		result.Content = []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Failed to ensure container running: %v", err)},
		}
		result.IsError = true
		return
	}

	// Get or create attached session for this container
	session, err := getOrCreateAttachedSession(ctx, cli, containerID)
	if err != nil {
		result.Content = []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Failed to attach to container: %v", err)},
		}
		result.IsError = true
		return
	}

	// Build the command string
	cmdString := args.Command
	if len(args.Argument) > 0 {
		cmdString = args.Command + " " + strings.Join(args.Argument, " ")
	}

	// Execute command and capture output via the attached session
	cmdResult, err := session.executeCommand(ctx, cli, containerID, cmdString)
	if err != nil {
		result.Content = []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Failed to execute command: %v", err)},
		}
		result.IsError = true
		return
	}

	// Return appropriate content based on result type
	if cmdResult.ImageData != "" {
		// Command timed out - return image of terminal state
		result.Content = []mcp.Content{
			&mcp.TextContent{Text: "[TIMEOUT] Command did not complete within 2 minutes. Terminal state captured below. Use 'showLastNExecution N' to see more history."},
			&mcp.ImageContent{
				MIMEType: "image/png",
				Data:     []byte(cmdResult.ImageData),
			},
		}
	} else {
		// Command completed - return text output
		result.Content = []mcp.Content{
			&mcp.TextContent{Text: cmdResult.Text},
		}
	}

	return
}

// ensureContainerRunning ensures the Kali container is running and returns its ID
func ensureContainerRunning(ctx context.Context, cli *client.Client) (string, error) {
	containerSummaries, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return "", fmt.Errorf("failed to list containers: %w", err)
	}

	var existingContainer *container.Summary
	for _, containerSummary := range containerSummaries {
		if containerSummary.Image == CONTAINER_IMAGE {
			existingContainer = &containerSummary
			break
		}
	}

	if existingContainer != nil {
		containerID := existingContainer.ID
		state := existingContainer.State

		switch state {
		case "created":
			log.Printf("Starting created container: %s", containerID)
			if err = cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
				return "", fmt.Errorf("failed to start created container: %w", err)
			}
			// Wait for tmux to initialize
			time.Sleep(2 * time.Second)
		case "paused":
			log.Printf("Unpausing paused container: %s", containerID)
			if err = cli.ContainerUnpause(ctx, containerID); err != nil {
				return "", fmt.Errorf("failed to unpause container: %w", err)
			}
		case "exited":
			log.Printf("Restarting exited container: %s", containerID)
			if err = cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
				return "", fmt.Errorf("failed to restart exited container: %w", err)
			}
			// Wait for tmux to initialize
			time.Sleep(2 * time.Second)
			// Invalidate old session since container restarted
			attachedSessionsLock.Lock()
			if session, exists := attachedSessions[containerID]; exists {
				session.Close()
				delete(attachedSessions, containerID)
			}
			attachedSessionsLock.Unlock()
		case "running":
			// Already running, good to go
		}

		return containerID, nil
	}

	// Create new container
	log.Printf("Creating new Kali container")
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image:     CONTAINER_IMAGE,
		Tty:       true,
		OpenStdin: true,
		StdinOnce: false,
	}, &container.HostConfig{
		CapAdd: []string{"NET_ADMIN", "NET_RAW"},
	}, nil, nil, "")
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	containerID := resp.ID

	if err = cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}
	log.Printf("Created and started new container: %s", containerID)

	// Wait for entrypoint.sh to set up tmux session
	time.Sleep(3 * time.Second)

	return containerID, nil
}

// getOrCreateAttachedSession returns an existing attached session or creates a new one
func getOrCreateAttachedSession(ctx context.Context, cli *client.Client, containerID string) (*AttachedSession, error) {
	attachedSessionsLock.RLock()
	session, exists := attachedSessions[containerID]
	attachedSessionsLock.RUnlock()

	if exists && session.isAlive() {
		return session, nil
	}

	attachedSessionsLock.Lock()
	defer attachedSessionsLock.Unlock()

	// Double-check after acquiring write lock
	if session, exists = attachedSessions[containerID]; exists && session.isAlive() {
		return session, nil
	}

	// Clean up old session if it exists
	if exists {
		session.Close()
	}

	// Create new attached session using docker attach
	sessionCtx, cancel := context.WithCancel(context.Background())

	attachResp, err := cli.ContainerAttach(sessionCtx, containerID, container.AttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to attach to container: %w", err)
	}

	session = &AttachedSession{
		containerID:  containerID,
		hijackedResp: attachResp,
		ctx:          sessionCtx,
		cancel:       cancel,
	}

	// Wait a bit for the attach to stabilize
	time.Sleep(300 * time.Millisecond)

	attachedSessions[containerID] = session
	log.Printf("Created new attached session to container: %s", containerID)

	return session, nil
}

// isAlive checks if the attached session is still active
func (s *AttachedSession) isAlive() bool {
	select {
	case <-s.ctx.Done():
		return false
	default:
		return true
	}
}

// Close closes the attached session
func (s *AttachedSession) Close() {
	s.cancel()
	s.hijackedResp.Close()
}

// CommandResult holds the result of command execution
type CommandResult struct {
	Text      string // Text output (if completed within timeout)
	ImageData string // Base64 encoded PNG (if timed out)
	TimedOut  bool   // Whether the command timed out
}

// executeCommand sends a command via the attached session (docker attach stdin)
// and reads output using tmux capture-pane (since tmux renders to TUI, not stdout)
func (s *AttachedSession) executeCommand(ctx context.Context, cli *client.Client, containerID string, command string) (*CommandResult, error) {
	// Capture the pane content BEFORE sending the command to know what's new
	beforeOutput, err := capturePaneContent(ctx, cli, containerID)
	if err != nil {
		log.Printf("Warning: failed to capture pane before command: %v", err)
		beforeOutput = ""
	}
	beforeLineCount := len(strings.Split(beforeOutput, "\n"))

	// Send command via the hijacked connection (like typing into terminal)
	_, err = io.WriteString(s.hijackedResp.Conn, command+"\n")
	if err != nil {
		return nil, fmt.Errorf("failed to write command: %w", err)
	}

	// Wait for command to complete by detecting prompt reappearance
	startTime := time.Now()
	maxWaitTime := 2 * time.Minute
	pollInterval := 500 * time.Millisecond

	// Give initial time for command to start executing
	time.Sleep(300 * time.Millisecond)

	var lastOutput string

	for {
		// Check if we've exceeded the timeout
		if time.Since(startTime) > maxWaitTime {
			log.Printf("Command timed out after 2 minutes, capturing visual state")
			// Capture visual state using screenshotter window via docker exec
			imageData, captureErr := captureVisualState(ctx, cli, containerID, 1)
			if captureErr != nil {
				log.Printf("Failed to capture visual state: %v", captureErr)
				// Return partial output with timeout message
				return &CommandResult{
					Text:     cleanOutput(lastOutput, command) + "\n\n[TIMEOUT: Command did not complete within 2 minutes. Use showLastNExecution to see current state.]",
					TimedOut: true,
				}, nil
			}
			return &CommandResult{
				ImageData: imageData,
				TimedOut:  true,
			}, nil
		}

		// Read output using tmux capture-pane (since tmux renders to TUI, not stdout)
		currentOutput, err := capturePaneContent(ctx, cli, containerID)
		if err != nil {
			log.Printf("Failed to capture pane content: %v", err)
			time.Sleep(pollInterval)
			continue
		}

		lastOutput = currentOutput

		// Look for prompt at the end of output (command completed)
		lines := strings.Split(currentOutput, "\n")

		// Find the last non-empty line
		var lastNonEmptyLine string
		for i := len(lines) - 1; i >= 0; i-- {
			trimmed := strings.TrimSpace(lines[i])
			if trimmed != "" {
				lastNonEmptyLine = trimmed
				break
			}
		}

		// Check if the last non-empty line is just the prompt (command finished)
		if promptRegex.MatchString(lastNonEmptyLine) {
			// Prompt found at end - command completed
			log.Printf("Command completed (prompt detected)")

			// Extract only the NEW output (lines added after the command was sent)
			newOutput := extractNewOutput(currentOutput, beforeLineCount, command)
			cleanedOutput := cleanOutput(newOutput, command)

			return &CommandResult{
				Text:     cleanedOutput,
				TimedOut: false,
			}, nil
		}

		// Small sleep to avoid busy waiting
		time.Sleep(pollInterval)
	}
}

// extractNewOutput extracts only the lines added after a command was executed
// It finds the LAST occurrence of the command echo (in case the same command was run before)
// and returns the output between that and the final prompt
func extractNewOutput(fullOutput string, beforeLineCount int, command string) string {
	lines := strings.Split(fullOutput, "\n")

	// Find the LAST occurrence of the command in the output (most recent execution)
	commandStartIdx := -1
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.Contains(lines[i], command) {
			commandStartIdx = i
			break
		}
	}

	if commandStartIdx == -1 {
		// Command not found, return full output
		return fullOutput
	}

	// Take lines from after the command echo to the end
	// (the command line itself contains "KALI[path]# <command>", skip it)
	if commandStartIdx+1 >= len(lines) {
		return ""
	}

	newLines := lines[commandStartIdx+1:]
	return strings.Join(newLines, "\n")
}

// capturePaneContent captures the current content of the agent tmux pane using docker exec
func capturePaneContent(ctx context.Context, cli *client.Client, containerID string) (string, error) {
	captureCmd := []string{"tmux", "capture-pane", "-pt", fmt.Sprintf("%s:%s", TMUX_SESSION, AGENT_WINDOW), "-S", "-50"}

	execConfig := container.ExecOptions{
		Cmd:          captureCmd,
		AttachStdout: true,
		AttachStderr: true,
	}

	execID, err := cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create exec for capture-pane: %w", err)
	}

	resp, err := cli.ContainerExecAttach(ctx, execID.ID, container.ExecAttachOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to attach to exec for capture-pane: %w", err)
	}
	defer resp.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, resp.Reader)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to read capture-pane output: %w", err)
	}

	output := buf.String()

	// Docker multiplexes stdout/stderr with 8-byte headers when TTY is false
	// Since our exec doesn't use TTY, we may get header bytes - strip them
	// Header format: [stream_type(1)][0][0][0][size(4)]
	if len(output) > 8 && (output[0] == 1 || output[0] == 2) {
		// Skip the 8-byte header
		output = output[8:]
	}

	return output, nil
}

// captureVisualState captures the terminal visual state using the screenshotter window
// This uses docker exec with tmux send-keys (only for screenshotter, not agent)
func captureVisualState(ctx context.Context, cli *client.Client, containerID string, n int) (string, error) {
	// Execute showLastNExecution in the screenshotter tmux window
	captureCmd := fmt.Sprintf("showLastNExecution %d", n)

	sendKeysCmd := []string{"tmux", "send-keys", "-t", fmt.Sprintf("%s:%s", TMUX_SESSION, SCREENSHOTTER_WINDOW), captureCmd, "Enter"}

	execConfig := container.ExecOptions{
		Cmd:          sendKeysCmd,
		AttachStdout: true,
		AttachStderr: true,
	}

	execID, err := cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create exec for capture: %w", err)
	}

	err = cli.ContainerExecStart(ctx, execID.ID, container.ExecStartOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to start exec for capture: %w", err)
	}

	// Wait for the screenshot to be generated
	time.Sleep(3 * time.Second)

	// Read the generated image file
	imageFile := fmt.Sprintf("/tmp/last_%d_cmds.png", n)

	// Use docker cp to get the file content
	reader, _, err := cli.CopyFromContainer(ctx, containerID, imageFile)
	if err != nil {
		return "", fmt.Errorf("failed to copy image from container: %w", err)
	}
	defer reader.Close()

	// Read tar archive (docker cp returns tar)
	var imageData bytes.Buffer
	_, err = io.Copy(&imageData, reader)
	if err != nil {
		return "", fmt.Errorf("failed to read image data: %w", err)
	}

	// Extract PNG from tar
	tarData := imageData.Bytes()
	if len(tarData) <= 512 {
		return "", fmt.Errorf("invalid tar data from container")
	}

	// Find PNG start (PNG magic bytes: 0x89 0x50 0x4E 0x47)
	pngStart := -1
	for i := 0; i < len(tarData)-4; i++ {
		if tarData[i] == 0x89 && tarData[i+1] == 0x50 && tarData[i+2] == 0x4E && tarData[i+3] == 0x47 {
			pngStart = i
			break
		}
	}
	if pngStart == -1 {
		return "", fmt.Errorf("PNG data not found in tar archive")
	}

	pngData := tarData[pngStart:]

	// Base64 encode the image
	base64Data := base64.StdEncoding.EncodeToString(pngData)
	return base64Data, nil
}

// cleanOutput removes the command echo, ANSI codes, and bash prompts from the output
func cleanOutput(output, command string) string {
	// Remove ANSI escape sequences (colors, cursor movements, bracketed paste mode, etc.)
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

		// Skip bash prompts (our custom prompt is "KALI[path]# ")
		if promptRegex.MatchString(trimmed) || strings.HasPrefix(trimmed, "KALI[") {
			continue
		}

		cleanedLines = append(cleanedLines, trimmed)
	}

	return strings.TrimSpace(strings.Join(cleanedLines, "\n"))
}
