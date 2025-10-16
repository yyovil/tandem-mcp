#!/usr/bin/env bash
set -euo pipefail

echo "🐳 Checking Docker status..."

# Check if Docker daemon is running
if ! docker info &> /dev/null; then
  echo "⚠️  Docker daemon is not running. Starting Docker Desktop..."

  # Start Docker Desktop if installed
  if [ -d "/Applications/Docker.app" ]; then
    open -a Docker
    echo "⏳ Waiting for Docker daemon to start..."

    # Wait for Docker to be ready (max 60 seconds)
    if timeout 60 sh -c 'until docker info &> /dev/null; do sleep 2; done'; then
      echo "✅ Docker daemon is now running"
    else
      echo "❌ Timeout waiting for Docker to start"
      echo "Please start Docker manually and try again"
    fi
  else
    echo "❌ Docker Desktop not found in /Applications"
    echo "Please start Docker manually"
  fi
else
  echo "✅ Docker daemon is running"
fi

# Inform about kali:headless image if Docker is running
if docker info &> /dev/null; then
  if ! docker image inspect kali:headless &> /dev/null; then
    echo "📦 kali:headless image not found"
  else
    echo "✅ kali:headless image is available"
  fi
fi

exit 0
