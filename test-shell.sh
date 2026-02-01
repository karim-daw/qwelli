#!/bin/bash

# Test script to demonstrate Qwelli shell commands
# This shows what commands are available in the interactive shell

set -e

echo "=== Qwelli Shell Test Guide ==="
echo ""
echo "The shell command provides an interactive REPL for Qwelli."
echo "Since it requires a TTY, you need to run it manually."
echo ""
echo "To test the shell:"
echo ""
echo "1. Make sure docker-compose is running:"
echo "   docker-compose ps"
echo ""
echo "2. Set your environment variables:"
echo "   export DATABASE_URL='postgresql://qwelli:qwelli@localhost:5432/qwelli?sslmode=disable'"
echo "   export VOYAGE_API_KEY='your_key'"
echo ""
echo "3. Start the shell:"
echo "   ./qwelli shell"
echo ""
echo "4. Try these commands in the shell:"
echo ""
echo "   Commands to test:"
echo "   ----------------"
echo "   help                          # Show all available commands"
echo "   list                          # List indexed folders"
echo "   use tests/demo/testdata       # Set current index"
echo "   search hello world            # Search in current index"
echo "   search python --top 3         # Limit results"
echo "   search cookies --strategy hybrid  # Use hybrid search"
echo "   model                         # Show current model"
echo "   clear                         # Clear screen"
echo "   exit                          # Exit shell"
echo ""
echo "=== Available Shell Commands ==="
echo ""

# Check if binary exists
if [ ! -f "./qwelli" ]; then
    echo "❌ Binary not found. Building..."
    go build -o qwelli ./cmd/qwelli
    echo "✅ Binary built"
fi

# Show help
echo "Getting shell help information..."
./qwelli shell --help

echo ""
echo "=== Quick Start ==="
echo ""
echo "Run this to start the shell:"
echo ""
echo "# The CLI automatically loads .env - just run:"
echo "./qwelli shell"
echo ""
echo "# Or if you need to override env vars:"
echo "DATABASE_URL='postgresql://...' ./qwelli shell"
