# Tandem MCP

An MCP (Model Context Protocol) server that provides a Docker-based terminal tool for executing commands in containers.

## Prerequisites

- Go 1.25.1 or later
- Docker installed and running

### Building

#### Build the MCP server

```bash
go build -v .
```

#### Build the docker image

```bash
cd kali && docker build -t kali:headless .
```

### Running the Server

use this cmd when asked while adding a new MCP server to an agent of your choice.

```bash
./tandem-mcp
```

### Available Tools

#### terminal

Executes a bash shell command in a Docker container (kali:headless).

#### show_last_n_execution

Captures a screenshot of the last N command executions from the agent window, useful for seeing terminal state after timeouts or during interactive sessions.

## License

Creative Commons Attribution-NonCommercial-ShareAlike 4.0 International
