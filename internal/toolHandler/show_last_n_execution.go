package toolhandler

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ShowLastNExecutionArgs defines the arguments for the ShowLastNExecution tool
type ShowLastNExecutionArgs struct {
	N int `json:"n" jsonschema:"Number of recent command executions to capture (default: 1),required"`
}

// ShowLastNExecution captures a screenshot of the last N command executions from the agent window
// This tool is useful for seeing terminal state, especially after timeouts or during interactive sessions
func ShowLastNExecution(ctx context.Context, request *mcp.CallToolRequest, args ShowLastNExecutionArgs) (result *mcp.CallToolResult, output any, err error) {
	result = &mcp.CallToolResult{}

	// Default to 1 if not specified or invalid
	n := args.N
	if n <= 0 {
		n = 1
	}

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

	// Capture visual state from the screenshotter window
	imageData, err := captureVisualState(ctx, cli, containerID, n)
	if err != nil {
		result.Content = []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Failed to capture visual state: %v", err)},
		}
		result.IsError = true
		return
	}

	result.Content = []mcp.Content{
		&mcp.TextContent{Text: fmt.Sprintf("Screenshot of last %d command execution(s) from the agent window:", n)},
		&mcp.ImageContent{
			MIMEType: "image/png",
			Data:     imageData,
		},
	}

	return
}
