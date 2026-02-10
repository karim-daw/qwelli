#!/bin/bash
set -e

echo "Building Qwelli Windows release (qwelli.exe + duckdb.dll)"
echo ""

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# 1. Install mingw-w64 if missing
if ! command -v x86_64-w64-mingw32-gcc &>/dev/null; then
    echo "Installing mingw-w64..."
    sudo apt-get update -qq
    sudo apt-get install -y gcc-mingw-w64-x86-64
fi

# 2. Build React frontend
echo "Building React frontend..."
cd web
if [ ! -d "node_modules" ]; then
    npm install
fi
npm run build
cd ..

# 3. Copy dist to embed location
rm -rf internal/server/web/dist
cp -r web/dist internal/server/web/

# 4. Download Windows DuckDB libraries (v1.4.3)
mkdir -p build/libs
DUCKDB_VERSION="v1.4.3"
DUCKDB_ZIP="/tmp/libduckdb-windows-amd64.zip"
echo "Downloading DuckDB Windows library ($DUCKDB_VERSION)..."
curl -sSL -o "$DUCKDB_ZIP" \
    "https://github.com/duckdb/duckdb/releases/download/${DUCKDB_VERSION}/libduckdb-windows-amd64.zip"
echo "Extracting..."
unzip -o -q "$DUCKDB_ZIP" -d build/libs/
rm -f "$DUCKDB_ZIP"

# 5. Cross-compile
echo "Cross-compiling for Windows (amd64)..."
CGO_ENABLED=1 \
GOOS=windows \
GOARCH=amd64 \
CC=x86_64-w64-mingw32-gcc \
CGO_CFLAGS="-I${ROOT}/build/libs" \
CGO_LDFLAGS="-L${ROOT}/build/libs -lduckdb" \
go build -ldflags "-H windowsgui" -o build/qwelli.exe ./cmd/qwelli

# 6. Package into zip
mkdir -p dist
cp build/qwelli.exe dist/
cp build/libs/duckdb.dll dist/
if command -v zip &>/dev/null; then
    cd dist && zip -q ../qwelli-windows-amd64.zip qwelli.exe duckdb.dll && cd ..
else
    python3 -c "
import zipfile
with zipfile.ZipFile('qwelli-windows-amd64.zip', 'w') as z:
    z.write('dist/qwelli.exe', 'qwelli.exe')
    z.write('dist/duckdb.dll', 'duckdb.dll')
"
fi

echo ""
echo "Build complete!"
echo "  dist/qwelli.exe"
echo "  dist/duckdb.dll"
echo "  qwelli-windows-amd64.zip"
echo ""
echo "Share the zip with your colleague. They extract it and double-click qwelli.exe."
