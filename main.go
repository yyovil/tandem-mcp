package main

import (
	"context"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yyovil/tandem-mcp/internal/prompts"
	toolhandler "github.com/yyovil/tandem-mcp/internal/toolHandler"
)

func main() {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "tandem mcp server for penetration testing",
		Version: "v0.0.1",
		Title:   "Tandem MCP",
	}, &mcp.ServerOptions{
		Instructions: prompts.Instructions,
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "Kali Terminal",
		Title:       "Kali Terminal",
		Description: prompts.Terminal,
	}, toolhandler.Terminal)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "ShowLastNExecution",
		Title:       "Show Last N Executions",
		Description: prompts.ShowLastNExecution,
	}, toolhandler.ShowLastNExecution)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
