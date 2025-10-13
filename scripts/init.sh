#!/bin/bash

set -e

# Check if module name is provided
if [ -z "$1" ]; then
    echo "Error: Module name is required"
    echo "Usage: ./scripts/init.sh {moduleName}"
    exit 1
fi

MODULE_NAME=$1

echo "Step 1: Removing remote branch tracking..."
git remote remove origin || echo "No origin remote found, skipping..."

echo "Step 2: Creating GitHub repository..."
gh repo create "$MODULE_NAME" --private --source=. --remote=origin --push

echo "Step 3: Allowing direnv..."
direnv allow

echo "Step 4: Updating Go module name..."
go mod edit -module "$MODULE_NAME"

echo "✓ Initialization complete!"