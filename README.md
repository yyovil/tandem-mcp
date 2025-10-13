# Tandem MCP

An MCP (Model Context Protocol) server that provides a Docker-based terminal tool for executing commands in containers.

## Features

- **MCP stdio transport protocol** - Implements the MCP protocol over stdio for seamless integration with MCP clients
- **Docker-based terminal tool** - Execute commands in a Kali Linux Docker container
- **Safe command execution** - Isolated execution environment using Docker containers
- **Automatic cleanup** - Containers are automatically removed after command execution

## Prerequisites

- Go 1.25.1 or later
- Docker installed and running
- Docker daemon accessible (default: `/var/run/docker.sock`)

## Installation

```bash
go build -o tandem-mcp .
```

## Usage

The server implements the MCP protocol and communicates via stdio. It's designed to be used by MCP clients.

### Running the Server

```bash
./tandem-mcp
```

### Available Tools

#### terminal

Execute commands in a Docker container (kalilinux/kali-rolling:latest).

**Parameters:**
- `command` (string, required): The command to execute
- `argument` (array of strings, optional): Arguments to pass to the command

**Example Request:**
```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "terminal",
    "arguments": {
      "command": "echo",
      "argument": ["Hello", "World"]
    }
  },
  "id": 1
}
```

**Example Response:**
```json
{
  "jsonrpc": "2.0",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Hello World\n"
      }
    ]
  },
  "id": 1
}
```

## Development

### Project Structure

```
.
├── main.go             # Main application file with MCP server implementation
├── go.mod              # Go module file
├── go.sum              # Go dependencies checksums
└── README.md           # This file
```

### Building

```bash
go build -v .
```

### Dependencies

- `github.com/modelcontextprotocol/go-sdk` - Official MCP protocol implementation
- `github.com/docker/docker` - Docker client SDK

## How It Works

1. The server listens on stdio for MCP protocol messages
2. When the `terminal` tool is called:
   - Creates a Docker client connection
   - Pulls the `kalilinux/kali-rolling:latest` image (if not already present)
   - Creates a container with the specified command and arguments
   - Starts the container and waits for it to complete
   - Retrieves the stdout from the container logs
   - Cleans up by removing the container
   - Returns the stdout as the tool result

## Security Considerations

- Commands are executed in isolated Docker containers
- Each execution uses a fresh container that is removed after completion
- The Docker daemon must be accessible to the server
- Consider the security implications of pulling and running Docker images

## License

Creative Commons Attribution-NonCommercial-ShareAlike 4.0 International