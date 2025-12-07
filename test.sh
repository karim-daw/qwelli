#!/bin/bash

# Automated test script for Qwelli CLI
# This tests the full workflow

set -e

echo "🧪 Qwelli CLI Test Suite"
echo "========================"
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if OpenAI API key is set
if [ -z "$QWELLI_EMBEDDING_KEY" ]; then
    echo -e "${RED}❌ Error: QWELLI_EMBEDDING_KEY environment variable not set${NC}"
    echo ""
    echo "Please set your OpenAI API key:"
    echo "  export QWELLI_EMBEDDING_KEY='sk-...'"
    echo ""
    echo "Or run manual init:"
    echo "  ./qwelli init"
    exit 1
fi

echo "✅ OpenAI API key found"
echo ""

# Build the binary
echo "📦 Building qwelli..."
go build -o qwelli ./cmd/qwelli
echo -e "${GREEN}✅ Build successful${NC}"
echo ""

# Clean up previous test data
echo "🧹 Cleaning up previous test data..."
rm -rf ~/.qwelli/
echo ""

# Create config programmatically
echo "⚙️  Creating configuration..."
mkdir -p ~/.qwelli/indexes
cat > ~/.qwelli/config.yaml << EOF
embedding_provider: openai
api_key: $QWELLI_EMBEDDING_KEY
model: text-embedding-3-small
endpoint: https://api.openai.com/v1/embeddings
index_dir: $HOME/.qwelli/indexes
EOF
chmod 600 ~/.qwelli/config.yaml
echo -e "${GREEN}✅ Configuration created${NC}"
echo ""

# Test: Index
echo "📁 Test 1: Indexing test folder..."
./qwelli index tests/demo/testdata
echo -e "${GREEN}✅ Indexing completed${NC}"
echo ""

# Test: List
echo "📋 Test 2: Listing indexes..."
./qwelli list
echo -e "${GREEN}✅ List command works${NC}"
echo ""

# Test: Status
echo "📊 Test 3: Checking index status..."
./qwelli status --index tests/demo/testdata
echo -e "${GREEN}✅ Status command works${NC}"
echo ""

# Test: Search - Greetings
echo "🔍 Test 4: Searching for greetings..."
./qwelli search "hello greeting" --index tests/demo/testdata --top 3
echo -e "${GREEN}✅ Search test 1 passed${NC}"
echo ""

# Test: Search - Farewells
echo "🔍 Test 5: Searching for farewells..."
./qwelli search "goodbye farewell" --index tests/demo/testdata --top 3
echo -e "${GREEN}✅ Search test 2 passed${NC}"
echo ""

# Test: Search - Technical
echo "🔍 Test 6: Searching for technical content..."
./qwelli search "machine learning neural networks" --index tests/demo/testdata --top 3
echo -e "${GREEN}✅ Search test 3 passed${NC}"
echo ""

# Test: Search - Code
echo "🔍 Test 7: Searching for code..."
./qwelli search "python function recursion" --index tests/demo/testdata --top 3
echo -e "${GREEN}✅ Search test 4 passed${NC}"
echo ""

# Test: Search - Recipe
echo "🔍 Test 8: Searching for recipes..."
./qwelli search "chocolate chip cookie recipe" --index tests/demo/testdata --top 3
echo -e "${GREEN}✅ Search test 5 passed${NC}"
echo ""

# Summary
echo "================================"
echo -e "${GREEN}✅ All tests passed!${NC}"
echo "================================"
echo ""
echo "📊 Summary:"
echo "  - Configuration: ✅"
echo "  - Indexing: ✅"
echo "  - Searching: ✅"
echo "  - List/Status: ✅"
echo ""
echo "💾 Test data stored in:"
echo "  ~/.qwelli/config.yaml"
echo "  ~/.qwelli/indexes/testdata.db"
echo ""
echo "🧹 To clean up:"
echo "  rm -rf ~/.qwelli/"
