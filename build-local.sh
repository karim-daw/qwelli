#!/bin/bash

# Build script for local platform only

VERSION="0.1.0"
BUILD_DIR="build"

echo "🔨 Building Qwelli v${VERSION} for current platform..."

# Clean previous builds
mkdir -p ${BUILD_DIR}

# Build for current platform
echo "📦 Building..."
go build -ldflags="-s -w" -o ${BUILD_DIR}/qwelli ./cmd/qwelli

if [ $? -eq 0 ]; then
    echo ""
    echo "✅ Build complete!"
    echo ""
    ls -lh ${BUILD_DIR}/qwelli
    echo ""
    echo "Run with: ./build/qwelli"
else
    echo "❌ Build failed!"
    exit 1
fi
