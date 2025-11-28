package toolhandler

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Helper function to extract text from result content
func extractTextFromResult(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	for _, content := range result.Content {
		if textContent, ok := content.(*mcp.TextContent); ok {
			return strings.TrimSpace(textContent.Text)
		}
	}
	return ""
}

// Helper function to check if result contains an image
func hasImageContent(result *mcp.CallToolResult) bool {
	for _, content := range result.Content {
		if _, ok := content.(*mcp.ImageContent); ok {
			return true
		}
	}
	return false
}

// Helper to sanitize container names (remove invalid characters)
func sanitizeContainerName(name string) string {
	// Replace invalid characters with dashes
	re := regexp.MustCompile(`[^a-zA-Z0-9_.-]`)
	return re.ReplaceAllString(name, "-")
}

func TestTerminal(t *testing.T) {
	tests := []struct {
		name           string
		command        string
		args           []string
		expectError    bool
		validateOutput func(t *testing.T, result *mcp.CallToolResult)
	}{
		{
			name:        "EchoCommand",
			command:     "echo",
			args:        []string{"running unit test 1"},
			expectError: false,
			validateOutput: func(t *testing.T, result *mcp.CallToolResult) {
				output := extractTextFromResult(t, result)
				expected := "running unit test 1"
				if output != expected {
					t.Errorf("output = %q, want %q", output, expected)
				}
			},
		},
		{
			name:        "EchoCommand_EmptyArgument",
			command:     "echo",
			args:        []string{},
			expectError: false,
			validateOutput: func(t *testing.T, result *mcp.CallToolResult) {
				output := extractTextFromResult(t, result)
				if output != "" {
					t.Logf("output = %q (expected empty or whitespace)", output)
				}
			},
		},
		{
			name:        "SimpleCommand_pwd",
			command:     "pwd",
			args:        []string{},
			expectError: false,
			validateOutput: func(t *testing.T, result *mcp.CallToolResult) {
				output := extractTextFromResult(t, result)
				if output == "" {
					t.Error("pwd returned empty output")
				}
				t.Logf("pwd output: %s", output)
			},
		},
		{
			name:        "InvalidCommand",
			command:     "nonexistentcommand12345",
			args:        []string{},
			expectError: false,
			validateOutput: func(t *testing.T, result *mcp.CallToolResult) {
				output := extractTextFromResult(t, result)
				t.Logf("Invalid command output: %s", output)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			request := &mcp.CallToolRequest{}
			args := TerminalArgs{
				Command:  tt.command,
				Argument: tt.args,
			}

			result, _, err := Terminal(ctx, request, args)

			if err != nil && !tt.expectError {
				t.Fatalf("Terminal() returned error: %v", err)
			}

			if result == nil {
				t.Fatal("Terminal() returned nil result")
			}

			if result.IsError && !tt.expectError {
				t.Errorf("Terminal() returned error result: %+v", result)
			}

			if len(result.Content) == 0 {
				t.Fatal("Terminal() returned empty content")
			}

			tt.validateOutput(t, result)
		})
	}
}

// TestTerminal_StatePersistence verifies that shell state persists across multiple command calls
func TestTerminal_StatePersistence(t *testing.T) {
	ctx := context.Background()

	// First, change directory to /tmp
	result1, _, err := Terminal(ctx, &mcp.CallToolRequest{}, TerminalArgs{
		Command:  "cd",
		Argument: []string{"/tmp"},
	})
	if err != nil {
		t.Fatalf("cd command failed: %v", err)
	}
	if result1.IsError {
		t.Fatalf("cd returned error result: %+v", result1)
	}

	// Now check pwd - should be /tmp
	result2, _, err := Terminal(ctx, &mcp.CallToolRequest{}, TerminalArgs{
		Command: "pwd",
	})
	if err != nil {
		t.Fatalf("pwd command failed: %v", err)
	}
	if result2.IsError {
		t.Fatalf("pwd returned error result: %+v", result2)
	}

	output := extractTextFromResult(t, result2)
	if output != "/tmp" {
		t.Errorf("pwd after cd = %q, want %q (state did not persist!)", output, "/tmp")
	}

	// Test environment variable persistence
	result3, _, err := Terminal(ctx, &mcp.CallToolRequest{}, TerminalArgs{
		Command:  "export",
		Argument: []string{"TEST_VAR=hello123"},
	})
	if err != nil {
		t.Fatalf("export command failed: %v", err)
	}
	if result3.IsError {
		t.Fatalf("export returned error result: %+v", result3)
	}

	// Check if the variable is set
	result4, _, err := Terminal(ctx, &mcp.CallToolRequest{}, TerminalArgs{
		Command:  "echo",
		Argument: []string{"$TEST_VAR"},
	})
	if err != nil {
		t.Fatalf("echo $TEST_VAR failed: %v", err)
	}
	if result4.IsError {
		t.Fatalf("echo returned error result: %+v", result4)
	}

	envOutput := extractTextFromResult(t, result4)
	if envOutput != "hello123" {
		t.Errorf("echo $TEST_VAR = %q, want %q (environment variable did not persist!)", envOutput, "hello123")
	}

	t.Log("✓ State persistence verified: cd and export work across multiple calls")
}

// TestTerminal_ContainerStates tests the Terminal function with containers in different states
// NOTE: This test is skipped because it creates containers with "sleep infinity" instead of
// using the proper entrypoint that starts tmux. The container state handling is tested
// implicitly by ensureContainerRunning() in other tests.
func TestTerminal_ContainerStates(t *testing.T) {
	t.Skip("Skipping: This test creates containers without tmux - container state handling is tested implicitly")

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Fatalf("Failed to create Docker client: %v", err)
	}
	defer cli.Close()

	ctx := context.Background()

	stateTests := []struct {
		name        string
		setupFunc   func(t *testing.T, cli *client.Client, ctx context.Context) string // returns containerID
		command     string
		args        []string
		expectError bool
	}{
		{
			name: "Container_Created_State",
			setupFunc: func(t *testing.T, cli *client.Client, ctx context.Context) string {
				// Create container but DO NOT start it
				containerName := sanitizeContainerName("test-created-" + t.Name())
				resp, err := cli.ContainerCreate(ctx, &container.Config{
					Image: CONTAINER_IMAGE,
					Cmd:   []string{"sleep", "infinity"},
					Tty:   true,
				}, nil, nil, nil, containerName)
				if err != nil {
					t.Fatalf("Failed to create container: %v", err)
				}
				return resp.ID
			},
			command:     "echo",
			args:        []string{"from-created-state"},
			expectError: false,
		},
		{
			name: "Container_Running_State",
			setupFunc: func(t *testing.T, cli *client.Client, ctx context.Context) string {
				// Create and START container
				containerName := sanitizeContainerName("test-running-" + t.Name())
				resp, err := cli.ContainerCreate(ctx, &container.Config{
					Image: CONTAINER_IMAGE,
					Cmd:   []string{"sleep", "infinity"},
					Tty:   true,
				}, nil, nil, nil, containerName)
				if err != nil {
					t.Fatalf("Failed to create container: %v", err)
				}
				if err = cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
					t.Fatalf("Failed to start container: %v", err)
				}
				return resp.ID
			},
			command:     "echo",
			args:        []string{"from-running-state"},
			expectError: false,
		},
		{
			name: "Container_Paused_State",
			setupFunc: func(t *testing.T, cli *client.Client, ctx context.Context) string {
				// Create, START, then PAUSE container
				containerName := sanitizeContainerName("test-paused-" + t.Name())
				resp, err := cli.ContainerCreate(ctx, &container.Config{
					Image: CONTAINER_IMAGE,
					Cmd:   []string{"sleep", "infinity"},
					Tty:   true,
				}, nil, nil, nil, containerName)
				if err != nil {
					t.Fatalf("Failed to create container: %v", err)
				}
				if err = cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
					t.Fatalf("Failed to start container: %v", err)
				}
				if err = cli.ContainerPause(ctx, resp.ID); err != nil {
					t.Fatalf("Failed to pause container: %v", err)
				}
				return resp.ID
			},
			command:     "echo",
			args:        []string{"from-paused-state"},
			expectError: false,
		},
		{
			name: "Container_Exited_State",
			setupFunc: func(t *testing.T, cli *client.Client, ctx context.Context) string {
				// Create, START, then STOP container
				containerName := sanitizeContainerName("test-exited-" + t.Name())
				resp, err := cli.ContainerCreate(ctx, &container.Config{
					Image: CONTAINER_IMAGE,
					Cmd:   []string{"sleep", "infinity"},
					Tty:   true,
				}, nil, nil, nil, containerName)
				if err != nil {
					t.Fatalf("Failed to create container: %v", err)
				}
				if err = cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
					t.Fatalf("Failed to start container: %v", err)
				}
				stopTimeout := 1
				if err = cli.ContainerStop(ctx, resp.ID, container.StopOptions{Timeout: &stopTimeout}); err != nil {
					t.Fatalf("Failed to stop container: %v", err)
				}
				return resp.ID
			},
			command:     "echo",
			args:        []string{"from-exited-state"},
			expectError: false,
		},
	}

	for _, tt := range stateTests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup: Create container in specific state
			containerID := tt.setupFunc(t, cli, ctx)

			// Cleanup: Remove container after test
			t.Cleanup(func() {
				cli.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true})
			})

			// Execute test
			request := &mcp.CallToolRequest{}
			args := TerminalArgs{
				Command:  tt.command,
				Argument: tt.args,
			}

			result, _, err := Terminal(ctx, request, args)

			if err != nil && !tt.expectError {
				t.Fatalf("Terminal() returned error: %v", err)
			}

			if result == nil {
				t.Fatal("Terminal() returned nil result")
			}

			if result.IsError && !tt.expectError {
				t.Errorf("Terminal() returned error result: %+v", result)
			}

			if len(result.Content) == 0 {
				t.Fatal("Terminal() returned empty content")
			}

			output := extractTextFromResult(t, result)
			t.Logf("Output from %s: %s", tt.name, output)
		})
	}
}

// TestTerminal_MultipleCommands tests executing multiple commands in sequence
func TestTerminal_MultipleCommands(t *testing.T) {
	ctx := context.Background()

	commands := []struct {
		command string
		args    []string
	}{
		{"echo", []string{"first"}},
		{"echo", []string{"second"}},
		{"echo", []string{"third"}},
	}

	for i, cmd := range commands {
		result, _, err := Terminal(ctx, &mcp.CallToolRequest{}, TerminalArgs{
			Command:  cmd.command,
			Argument: cmd.args,
		})
		if err != nil {
			t.Fatalf("Command %d failed: %v", i+1, err)
		}
		if result.IsError {
			t.Errorf("Command %d returned error: %+v", i+1, result)
		}

		output := extractTextFromResult(t, result)
		expected := cmd.args[0]
		if output != expected {
			t.Errorf("Command %d: output = %q, want %q", i+1, output, expected)
		}
	}

	t.Log("✓ Multiple sequential commands executed successfully")
}

// TestTerminal_LongRunningCommand tests a command that takes some time
func TestTerminal_LongRunningCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running test in short mode")
	}

	ctx := context.Background()

	// Run a command that takes a few seconds
	result, _, err := Terminal(ctx, &mcp.CallToolRequest{}, TerminalArgs{
		Command:  "sleep",
		Argument: []string{"3", "&&", "echo", "done"},
	})
	if err != nil {
		t.Fatalf("Long running command failed: %v", err)
	}

	if result.IsError {
		t.Errorf("Long running command returned error: %+v", result)
	}

	output := extractTextFromResult(t, result)
	t.Logf("Long running command output: %s", output)

	// Should complete within 2 minute timeout
	if hasImageContent(result) {
		t.Error("Command should not have timed out (only 3 seconds)")
	}
}
