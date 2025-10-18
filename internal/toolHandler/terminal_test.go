package toolhandler

import (
	"context"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestTerminal(t *testing.T) {
	tests := []struct {
		name           string
		command        string
		args           []string
		expectError    bool
		validateOutput func(t *testing.T, output string)
	}{
		{
			name:        "EchoCommand",
			command:     "echo",
			args:        []string{"running unit test 1"},
			expectError: false,
			validateOutput: func(t *testing.T, output string) {
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
			validateOutput: func(t *testing.T, output string) {
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
			validateOutput: func(t *testing.T, output string) {
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
			validateOutput: func(t *testing.T, output string) {
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

			textContent, ok := result.Content[0].(*mcp.TextContent)
			if !ok {
				t.Fatalf("content is not TextContent, got: %T", result.Content[0])
			}

			output := strings.TrimSpace(textContent.Text)
			tt.validateOutput(t, output)
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

	textContent, ok := result2.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content is not TextContent")
	}

	output := strings.TrimSpace(textContent.Text)
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

	textContent2, ok := result4.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content is not TextContent")
	}

	envOutput := strings.TrimSpace(textContent2.Text)
	if envOutput != "hello123" {
		t.Errorf("echo $TEST_VAR = %q, want %q (environment variable did not persist!)", envOutput, "hello123")
	}

	t.Log("✓ State persistence verified: cd and export work across multiple calls")
}

// TestTerminal_ContainerStates tests the Terminal function with containers in different states
func TestTerminal_ContainerStates(t *testing.T) {
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
				resp, err := cli.ContainerCreate(ctx, &container.Config{
					Image: CONTAINER_IMAGE,
					Cmd:   []string{"sleep", "infinity"},
					Tty:   true,
				}, nil, nil, nil, "test-created-"+t.Name())
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
				resp, err := cli.ContainerCreate(ctx, &container.Config{
					Image: CONTAINER_IMAGE,
					Cmd:   []string{"sleep", "infinity"},
					Tty:   true,
				}, nil, nil, nil, "test-running-"+t.Name())
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
				resp, err := cli.ContainerCreate(ctx, &container.Config{
					Image: CONTAINER_IMAGE,
					Cmd:   []string{"sleep", "infinity"},
					Tty:   true,
				}, nil, nil, nil, "test-paused-"+t.Name())
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
				resp, err := cli.ContainerCreate(ctx, &container.Config{
					Image: CONTAINER_IMAGE,
					Cmd:   []string{"sleep", "infinity"},
					Tty:   true,
				}, nil, nil, nil, "test-exited-"+t.Name())
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

			textContent, ok := result.Content[0].(*mcp.TextContent)
			if !ok {
				t.Fatalf("content is not TextContent, got: %T", result.Content[0])
			}

			output := strings.TrimSpace(textContent.Text)
			t.Logf("Output from %s: %s", tt.name, output)
		})
	}
}
