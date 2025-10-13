package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// Create MCP server
	mcpServer := server.NewMCPServer(
		"tandem-mcp",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// Register terminal tool
	terminalTool := mcp.NewTool("terminal",
		mcp.WithDescription("Execute commands in a Docker container"),
		mcp.WithString("command",
			mcp.Description("The command to execute"),
			mcp.Required(),
		),
		mcp.WithArray("argument",
			mcp.Description("Arguments for the command"),
			mcp.Items(map[string]interface{}{
				"type": "string",
			}),
		),
	)

	// Add tool handler
	mcpServer.AddTool(terminalTool, executeTerminalCommand)

	// Start server with stdio transport
	if err := server.ServeStdio(mcpServer); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func executeTerminalCommand(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract command from arguments
	command := request.GetString("command", "")
	if command == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: command parameter is required and must be a string",
				},
			},
			IsError: true,
		}, nil
	}

	// Extract arguments (optional)
	var args []string
	arguments := request.GetArguments()
	if argsInterface, exists := arguments["argument"]; exists {
		if argsList, ok := argsInterface.([]interface{}); ok {
			for _, arg := range argsList {
				if argStr, ok := arg.(string); ok {
					args = append(args, argStr)
				}
			}
		}
	}

	// Create Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to create Docker client: %v", err),
				},
			},
			IsError: true,
		}, nil
	}
	defer cli.Close()

	// Pull kali:headless image if not present
	imageName := "kalilinux/kali-rolling:latest"
	
	// Check if image exists
	_, _, err = cli.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		// Image doesn't exist, try to pull it
		log.Printf("Pulling image %s...", imageName)
		reader, err := cli.ImagePull(ctx, imageName, image.PullOptions{})
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Failed to pull image: %v", err),
					},
				},
				IsError: true,
			}, nil
		}
		defer reader.Close()
		// Wait for pull to complete
		io.Copy(os.Stderr, reader)
	}

	// Build command with arguments
	cmdWithArgs := append([]string{command}, args...)

	// Create container
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: imageName,
		Cmd:   cmdWithArgs,
		Tty:   false,
	}, nil, nil, nil, "")
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to create container: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// Ensure cleanup
	defer func() {
		if err := cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true}); err != nil {
			log.Printf("Failed to remove container: %v", err)
		}
	}()

	// Start container
	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to start container: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// Wait for container to finish
	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error waiting for container: %v", err),
					},
				},
				IsError: true,
			}, nil
		}
	case <-statusCh:
	}

	// Get container logs (stdout)
	out, err := cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{ShowStdout: true})
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to get container logs: %v", err),
				},
			},
			IsError: true,
		}, nil
	}
	defer out.Close()

	// Read stdout
	stdout, err := io.ReadAll(out)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to read stdout: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// Return result
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(stdout),
			},
		},
	}, nil
}
