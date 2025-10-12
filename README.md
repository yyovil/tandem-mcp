# Go Project Template

A minimal Go project template with Nix devShell and direnv integration.

## Features

- 🚀 Go 1.25.1 ready
- ❄️ Nix flakes for reproducible development environment
- 🔧 direnv for automatic environment loading
- 📦 Pre-configured with essential Go tools (gopls, gotools, goreleaser)

## Prerequisites

- [Nix](https://nixos.org/download.html) with flakes enabled
- [direnv](https://direnv.net/) (optional but recommended)

### Enable Nix Flakes

Add to `~/.config/nix/nix.conf` (or `/etc/nix/nix.conf`):

```nix
experimental-features = nix-command flakes
```

## Getting Started

### 1. Use This Template

Click the "Use this template" button on GitHub or clone this repository:

```bash
git clone <your-repo-url>
cd <your-project-name>
```

### 2. Update Module Name

Edit `go.mod` and replace the module path with your own:

```go
module github.com/YOUR_USERNAME/YOUR_PROJECT
```

### 3. Enter Development Environment

#### With direnv (recommended):

```bash
direnv allow
```

The environment will automatically load when you `cd` into the project directory.

#### Without direnv:

```bash
nix develop
```

### 4. Verify Setup

```bash
go version
go list
```

## Project Structure

```
.
├── .envrc              # direnv configuration
├── .gitignore          # Git ignore rules
├── flake.nix           # Nix flake for development environment
├── flake.lock          # Locked dependencies
├── go.mod              # Go module file
├── go.sum              # Go dependencies checksums
├── main.go             # Main application file
└── pkgs/               # Additional packages directory
```

## Development

### Available Tools

The Nix development shell includes:

- `go` - Go compiler and toolchain
- `gopls` - Go language server
- `gotools` - Additional Go tools
- `goreleaser` - Release automation tool

### Building

```bash
go build -o bin/app .
```

### Running

```bash
go run main.go
```

### Testing

```bash
go test ./...
```

## Environment Variables

The development environment sets:

- `GOROOT` - Points to the Nix-managed Go installation
- `GOPATH` - Set to `.go` in the project directory
- `GOBIN` - Set to `.go/bin` for installed binaries
- `PATH` - Updated to include `$GOPATH/bin`

## Customization

### Modifying the Nix Environment

Edit `flake.nix` to add or remove packages in the `buildInputs` section:

```nix
buildInputs = with pkgs; [ 
  go 
  gopls 
  gotools 
  goreleaser
  # Add more packages here
];
```

## CI/CD

This template is ready for GitHub Actions or other CI/CD pipelines. The Nix flake ensures consistent builds across different environments.

## License

Specify your license here.