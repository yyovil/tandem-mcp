package toolhandler

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	dockerClient     *client.Client
	dockerClientOnce sync.Once
	dockerClientErr  error
)

const CONTAINER_IMAGE = "ghcr.io/yyovil/kali:headless"

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
			Image: CONTAINER_IMAGE,
			Cmd: []string{"sleep", "infinity"},
			Tty: true,
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

	execConfig := container.ExecOptions{
		Cmd:          append([]string{args.Command}, args.Argument...),
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err := cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		result.Content = []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Failed to create exec instance: %v", err)},
		}
		result.IsError = true
		return
	}

	attachResp, err := cli.ContainerExecAttach(ctx, execResp.ID, container.ExecStartOptions{})
	if err != nil {
		result.Content = []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Failed to attach to exec instance: %v", err)},
		}
		result.IsError = true
		return
	}
	defer attachResp.Close()

	// Use stdcopy.StdCopy to demultiplex the output stream
	// This removes the Docker stream headers
	outputBuffer := new(strings.Builder)
	_, err = stdcopy.StdCopy(outputBuffer, outputBuffer, attachResp.Reader)
	if err != nil {
		result.Content = []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Failed to read exec output: %v", err)},
		}
		result.IsError = true
		return
	}

	result.Content = []mcp.Content{
		&mcp.TextContent{Text: outputBuffer.String()},
	}
	return
}
