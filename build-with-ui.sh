#!/bin/bash
set -e

echo "🔨 Building Qwelli with Web UI"
echo ""

# Step 1: Build React frontend
echo "📦 Building React frontend..."
cd web

if [ ! -d "node_modules" ]; then
    echo "📥 Installing frontend dependencies..."
    npm install
fi

echo "🔨 Building frontend assets..."
npm run build

cd ..

# Step 2: Ensure the embed path exists
if [ ! -d "internal/server/web/dist" ]; then
    echo "📁 Creating embed directory..."
    mkdir -p internal/server/web
fi

# Copy dist to embed location
echo "📋 Copying built assets to embed location..."
rm -rf internal/server/web/dist
cp -r web/dist internal/server/web/

# Step 3: Build Go binary
echo "🔨 Building Go binary..."
go build -o qwelli ./cmd/qwelli

echo ""
echo "✅ Build complete!"
echo "   Binary: ./qwelli"
echo ""
echo "To start the web UI, run:"
echo "   ./qwelli serve"
