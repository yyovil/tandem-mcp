package toolhandler

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestShowLastNExecution tests the ShowLastNExecution tool
func TestShowLastNExecution(t *testing.T) {
	ctx := context.Background()

	// First run a few commands to have something to capture
	_, _, err := Terminal(ctx, &mcp.CallToolRequest{}, TerminalArgs{
		Command:  "echo",
		Argument: []string{"test command 1"},
	})
	if err != nil {
		t.Fatalf("First echo failed: %v", err)
	}

	_, _, err = Terminal(ctx, &mcp.CallToolRequest{}, TerminalArgs{
		Command:  "echo",
		Argument: []string{"test command 2"},
	})
	if err != nil {
		t.Fatalf("Second echo failed: %v", err)
	}

	// Now test ShowLastNExecution tool
	result, _, err := ShowLastNExecution(ctx, &mcp.CallToolRequest{}, ShowLastNExecutionArgs{
		N: 2,
	})
	if err != nil {
		t.Fatalf("ShowLastNExecution failed: %v", err)
	}

	if result.IsError {
		t.Errorf("ShowLastNExecution returned error: %+v", result)
	}

	// Should have both text and image content
	if len(result.Content) < 2 {
		t.Errorf("Expected at least 2 content items (text + image), got %d", len(result.Content))
	}

	// Check for image content
	if !hasImageContent(result) {
		t.Error("ShowLastNExecution should return an image")
	}

	// Log the text content
	textOutput := extractTextFromResult(t, result)
	t.Logf("ShowLastNExecution text: %s", textOutput)
}

// TestShowLastNExecution_DefaultN tests that N defaults to 1 when not specified or invalid
func TestShowLastNExecution_DefaultN(t *testing.T) {
	ctx := context.Background()

	// Run a command first
	_, _, err := Terminal(ctx, &mcp.CallToolRequest{}, TerminalArgs{
		Command:  "echo",
		Argument: []string{"default N test"},
	})
	if err != nil {
		t.Fatalf("Echo command failed: %v", err)
	}

	// Test with N=0 (should default to 1)
	result, _, err := ShowLastNExecution(ctx, &mcp.CallToolRequest{}, ShowLastNExecutionArgs{
		N: 0,
	})
	if err != nil {
		t.Fatalf("ShowLastNExecution with N=0 failed: %v", err)
	}

	if result.IsError {
		t.Errorf("ShowLastNExecution returned error: %+v", result)
	}

	if !hasImageContent(result) {
		t.Error("ShowLastNExecution should return an image")
	}

	// Test with negative N (should default to 1)
	result2, _, err := ShowLastNExecution(ctx, &mcp.CallToolRequest{}, ShowLastNExecutionArgs{
		N: -5,
	})
	if err != nil {
		t.Fatalf("ShowLastNExecution with N=-5 failed: %v", err)
	}

	if result2.IsError {
		t.Errorf("ShowLastNExecution returned error: %+v", result2)
	}

	if !hasImageContent(result2) {
		t.Error("ShowLastNExecution should return an image")
	}
}
