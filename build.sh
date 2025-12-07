#!/bin/bash

# Build script for cross-platform binaries
# NOTE: DuckDB uses CGo, so cross-compilation requires platform-specific toolchains.
# For production builds, use GitHub Actions or build on each target platform.

VERSION="0.1.0"
BUILD_DIR="build"

echo "🔨 Building Qwelli v${VERSION} for all platforms..."
echo "⚠️  Note: Cross-compilation may fail due to CGo dependencies"
echo ""

# Clean previous builds
rm -rf ${BUILD_DIR}
mkdir -p ${BUILD_DIR}

SUCCESS_COUNT=0
FAIL_COUNT=0

# Function to build for a platform
build_platform() {
    local OS=$1
    local ARCH=$2
    local OUTPUT=$3
    
    echo "📦 Building for ${OS}/${ARCH}..."
    if GOOS=${OS} GOARCH=${ARCH} CGO_ENABLED=1 go build -ldflags="-s -w" -o ${BUILD_DIR}/${OUTPUT} ./cmd/qwelli 2>&1; then
        echo "   ✅ Success"
        SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
    else
        echo "   ❌ Failed (CGo cross-compilation not supported)"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
    echo ""
}

# Try to build for all platforms
build_platform "windows" "amd64" "qwelli-windows-amd64.exe"
build_platform "darwin" "amd64" "qwelli-macos-amd64"
build_platform "darwin" "arm64" "qwelli-macos-arm64"
build_platform "linux" "amd64" "qwelli-linux-amd64"
build_platform "linux" "arm64" "qwelli-linux-arm64"

echo "================================="
echo "Build Summary:"
echo "  ✅ Successful: ${SUCCESS_COUNT}"
echo "  ❌ Failed: ${FAIL_COUNT}"
echo "================================="
echo ""

if [ ${SUCCESS_COUNT} -gt 0 ]; then
    echo "Successfully built binaries:"
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
else
    echo "❌ All builds failed!"
    echo ""
    echo "💡 For local development, use: ./build-local.sh"
    echo "💡 For production releases, use GitHub Actions (builds on each platform natively)"
    exit 1
fi
