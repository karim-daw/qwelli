#!/bin/bash
set -e

echo "============================================"
echo "  Qwelli Build"
echo "============================================"
echo ""

# Parse arguments
RELEASE=0
for arg in "$@"; do
    if [ "$arg" = "--release" ]; then
        RELEASE=1
    fi
done

# Get project root
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# ============================================
# Step 1: Check prerequisites
# ============================================
echo "[1/4] Checking prerequisites..."

if ! command -v go &>/dev/null; then
    echo "X Error: Go not found in PATH"
    exit 1
fi
echo "  + Go found"

if ! command -v node &>/dev/null; then
    echo "X Error: Node.js not found in PATH"
    exit 1
fi
echo "  + Node.js found"
echo ""

# ============================================
# Step 2: Build React frontend
# ============================================
echo "[2/4] Building React frontend..."

cd "$ROOT/web"

if [ ! -d "node_modules" ]; then
    echo "  Installing dependencies..."
    npm install --silent
fi

npm run build --silent

cd "$ROOT"
echo "  + Frontend built"
echo ""

# ============================================
# Step 3: Copy frontend to embed location
# ============================================
echo "[3/4] Copying frontend to embed location..."

rm -rf internal/server/web/dist
mkdir -p internal/server/web
cp -r web/dist internal/server/web/

echo "  + Done"
echo ""

# ============================================
# Step 4: Build Go binary
# ============================================
echo "[4/4] Building Go binary..."

mkdir -p build

if [ "$RELEASE" = "1" ]; then
    echo "  Mode: release"
    go build -ldflags "-s -w" -o build/qwelli ./cmd/qwelli
else
    echo "  Mode: dev"
    go build -o build/qwelli ./cmd/qwelli
fi

echo "  + build/qwelli"
echo ""

echo "============================================"
echo "  Build Complete!"
echo "============================================"
echo ""
echo "  Run: ./build/qwelli serve"
echo ""
