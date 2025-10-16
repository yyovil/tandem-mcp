# Tandem MCP

An MCP (Model Context Protocol) server that provides a Docker-based terminal tool for executing commands in containers.

## Prerequisites

- Go 1.25.1 or later
- Docker installed and running

### Building

```bash
go build -v .
```

### Running the Server

```bash
./tandem-mcp
```

### Available Tools

#### terminal

Executes a shell command in a Docker container (ghcr.io/yyovil/kali:headless).

### Project Structure

```text
.
├── main.go             # Main application file with MCP server implementation
├── internal/
│   ├── toolHandlers/   # Contains all tool handler implementations
│   ├── prompts/        # Contains prompts used throughout the MCP server
├── go.mod              # Go module file
├── go.sum              # Go dependencies checksums
└── README.md           # This file
```

## License

Creative Commons Attribution-NonCommercial-ShareAlike 4.0 International
