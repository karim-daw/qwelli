#!/bin/bash

# Demo script for Qwelli CLI
# This demonstrates the full workflow

set -e

echo "🚀 Qwelli CLI Demo"
echo "=================="
echo ""

# Build the binary
echo "📦 Building qwelli..."
go build -o qwelli ./cmd/qwelli
echo ""

# Show version
echo "✅ Built successfully!"
./qwelli --version
echo ""

# Show available commands
echo "📋 Available commands:"
./qwelli --help
echo ""

echo "🎯 To use Qwelli:"
echo ""
echo "1. Initialize configuration:"
echo "   ./qwelli init"
echo ""
echo "2. Index a folder:"
echo "   ./qwelli index tests/demo/testdata"
echo ""
echo "3. Search:"
echo "   ./qwelli search \"hello greeting\" --index tests/demo/testdata"
echo ""
echo "4. List indexes:"
echo "   ./qwelli list"
echo ""
echo "5. Check status:"
echo "   ./qwelli status --index tests/demo/testdata"
echo ""

echo "💡 Note: You'll need to run 'qwelli init' first and provide your OpenAI API key"
