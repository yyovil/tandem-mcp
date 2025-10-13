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
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TerminalArgs defines the input parameters for the terminal tool
type TerminalArgs struct {
	Command  string   `json:"command" jsonschema:"The command to execute,required"`
	Argument []string `json:"argument,omitempty" jsonschema:"Arguments for the command"`
}

func main() {
	// Create MCP server
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "tandem-mcp",
		Version: "1.0.0",
	}, nil)

	// Register terminal tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        "terminal",
		Description: "Execute commands in a Docker container",
	}, executeTerminalCommand)

	// Start server with stdio transport
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func executeTerminalCommand(ctx context.Context, request *mcp.CallToolRequest, args TerminalArgs) (*mcp.CallToolResult, any, error) {
	// Validate command
	if args.Command == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "Error: command parameter is required and must be a string"},
			},
			IsError: true,
		}, nil, nil
	}

	// Create Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Failed to create Docker client: %v", err)},
			},
			IsError: true,
		}, nil, nil
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
					&mcp.TextContent{Text: fmt.Sprintf("Failed to pull image: %v", err)},
				},
				IsError: true,
			}, nil, nil
		}
		defer reader.Close()
		// Wait for pull to complete
		io.Copy(os.Stderr, reader)
	}

	// Build command with arguments
	cmdWithArgs := append([]string{args.Command}, args.Argument...)

	// Create container
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: imageName,
		Cmd:   cmdWithArgs,
		Tty:   false,
	}, nil, nil, nil, "")
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Failed to create container: %v", err)},
			},
			IsError: true,
		}, nil, nil
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
				&mcp.TextContent{Text: fmt.Sprintf("Failed to start container: %v", err)},
			},
			IsError: true,
		}, nil, nil
	}

	// Wait for container to finish
	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Error waiting for container: %v", err)},
				},
				IsError: true,
			}, nil, nil
		}
	case <-statusCh:
	}

	// Get container logs (stdout)
	out, err := cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{ShowStdout: true})
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Failed to get container logs: %v", err)},
			},
			IsError: true,
		}, nil, nil
	}
	defer out.Close()

	// Read stdout
	stdout, err := io.ReadAll(out)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Failed to read stdout: %v", err)},
			},
			IsError: true,
		}, nil, nil
	}

	// Return result
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(stdout)},
		},
	}, nil, nil
}
