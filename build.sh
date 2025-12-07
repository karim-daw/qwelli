#!/bin/bash

# Build script for cross-platform binaries

VERSION="0.1.0"
BUILD_DIR="build"

echo "🔨 Building Qwelli v${VERSION} for all platforms..."

# Clean previous builds
rm -rf ${BUILD_DIR}
mkdir -p ${BUILD_DIR}

# Windows (64-bit)
echo "📦 Building for Windows (amd64)..."
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o ${BUILD_DIR}/qwelli-windows-amd64.exe ./cmd/qwelli

# macOS (Intel)
echo "📦 Building for macOS (amd64)..."
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o ${BUILD_DIR}/qwelli-macos-amd64 ./cmd/qwelli

# macOS (Apple Silicon)
echo "📦 Building for macOS (arm64)..."
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o ${BUILD_DIR}/qwelli-macos-arm64 ./cmd/qwelli

# Linux (64-bit)
echo "📦 Building for Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ${BUILD_DIR}/qwelli-linux-amd64 ./cmd/qwelli

# Linux (ARM64) - for Raspberry Pi, etc.
echo "📦 Building for Linux (arm64)..."
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o ${BUILD_DIR}/qwelli-linux-arm64 ./cmd/qwelli

echo ""
echo "✅ Build complete! Binaries in ${BUILD_DIR}/"
echo ""
ls -lh ${BUILD_DIR}/

echo ""
echo "📊 File sizes:"
du -sh ${BUILD_DIR}/*

echo ""
echo "📦 To create archives:"
echo "  cd ${BUILD_DIR}"
echo "  zip qwelli-windows-amd64.zip qwelli-windows-amd64.exe"
echo "  tar -czf qwelli-macos-amd64.tar.gz qwelli-macos-amd64"
echo "  tar -czf qwelli-linux-amd64.tar.gz qwelli-linux-amd64"
