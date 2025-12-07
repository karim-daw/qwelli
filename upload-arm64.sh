#!/bin/bash

# Script to build and upload ARM64 binaries to GitHub release
# Usage: ./upload-arm64.sh v0.1.0

set -e

if [ -z "$1" ]; then
    echo "Usage: $0 <tag>"
    echo "Example: $0 v0.1.0"
    exit 1
fi

TAG=$1
BUILD_DIR="build"

echo "🔨 Building ARM64 binaries for release ${TAG}..."
echo ""

# Detect current platform
GOOS=$(go env GOOS)
GOARCH=$(go env GOARCH)

echo "Current platform: ${GOOS}/${GOARCH}"
echo ""

# Build for current platform (assuming it's ARM64)
if [ "$GOARCH" != "arm64" ]; then
    echo "⚠️  Warning: You're not on ARM64. This script is for ARM64 builds."
    echo "Current arch: ${GOARCH}"
    read -p "Continue anyway? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

mkdir -p ${BUILD_DIR}

# Build based on OS
if [ "$GOOS" = "linux" ]; then
    OUTPUT="${BUILD_DIR}/qwelli-linux-arm64"
    echo "📦 Building ${OUTPUT}..."
    go build -ldflags="-s -w" -o ${OUTPUT} ./cmd/qwelli
    echo "✅ Built ${OUTPUT}"
elif [ "$GOOS" = "windows" ]; then
    OUTPUT="${BUILD_DIR}/qwelli-windows-arm64.exe"
    echo "📦 Building ${OUTPUT}..."
    go build -ldflags="-s -w" -o ${OUTPUT} ./cmd/qwelli
    echo "✅ Built ${OUTPUT}"
else
    echo "❌ Unsupported OS for ARM64 manual build: ${GOOS}"
    echo "This script supports Linux and Windows ARM64 only."
    echo "macOS ARM64 is built by GitHub Actions."
    exit 1
fi

echo ""
echo "📊 Binary size:"
ls -lh ${OUTPUT}
echo ""

# Check if gh CLI is installed
if ! command -v gh &> /dev/null; then
    echo "⚠️  GitHub CLI (gh) not found."
    echo ""
    echo "To upload to GitHub release, install gh CLI:"
    echo "  https://cli.github.com/"
    echo ""
    echo "Or manually upload ${OUTPUT} to:"
    echo "  https://github.com/YOUR_USERNAME/qwelli/releases/tag/${TAG}"
    exit 0
fi

# Upload to release
echo "📤 Uploading to GitHub release ${TAG}..."
gh release upload ${TAG} ${OUTPUT} --clobber

echo ""
echo "✅ Successfully uploaded ${OUTPUT} to release ${TAG}!"
echo ""
echo "View release: gh release view ${TAG} --web"
