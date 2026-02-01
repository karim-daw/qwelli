#!/bin/bash

# Automated test script for Qwelli
# Tests all major functionality with docker-compose running

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Qwelli Automated Test Suite ===${NC}"
echo ""

# Check prerequisites
echo "Checking prerequisites..."

if ! docker ps | grep -q qwelli-postgres; then
    echo -e "${RED}❌ Docker containers not running. Start with: docker-compose up -d${NC}"
    exit 1
fi

if [ ! -f "./qwelli" ]; then
    echo "Building binary..."
    go build -o qwelli ./cmd/qwelli
fi

# Environment variables are automatically loaded from .env file
# No need to export manually!

echo -e "${GREEN}✓ Prerequisites OK${NC}"
echo ""

# Test 1: Health Checks
echo -e "${BLUE}Test 1: Health Checks${NC}"
if curl -s http://localhost:8080/health | grep -q "healthy"; then
    echo -e "${GREEN}✓ API health endpoint working${NC}"
else
    echo -e "${RED}✗ API health endpoint failed${NC}"
    exit 1
fi

if curl -s http://localhost:8080/ready | grep -q "ready"; then
    echo -e "${GREEN}✓ API ready endpoint working${NC}"
else
    echo -e "${RED}✗ API ready endpoint failed${NC}"
    exit 1
fi
echo ""

# Test 2: Database
echo -e "${BLUE}Test 2: Database${NC}"
if docker exec qwelli-postgres psql -U qwelli -d qwelli -c "SELECT 1" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ PostgreSQL accessible${NC}"
else
    echo -e "${RED}✗ PostgreSQL connection failed${NC}"
    exit 1
fi

if docker exec qwelli-postgres psql -U qwelli -d qwelli -c "SELECT * FROM pg_extension WHERE extname='vector'" | grep -q "vector"; then
    echo -e "${GREEN}✓ pgvector extension installed${NC}"
else
    echo -e "${RED}✗ pgvector extension missing${NC}"
    exit 1
fi
echo ""

# Test 3: CLI List
echo -e "${BLUE}Test 3: CLI List Command${NC}"
if ./qwelli list > /dev/null 2>&1; then
    echo -e "${GREEN}✓ List command working${NC}"
else
    echo -e "${RED}✗ List command failed${NC}"
    exit 1
fi
echo ""

# Test 4: CLI Search (Semantic)
echo -e "${BLUE}Test 4: CLI Search (Semantic)${NC}"
SEARCH_OUTPUT=$(./qwelli search "hello world" --index tests/demo/testdata/ 2>&1)
if echo "$SEARCH_OUTPUT" | grep -q "Result 1"; then
    echo -e "${GREEN}✓ Semantic search working - returned results${NC}"
else
    echo -e "${RED}✗ Semantic search failed${NC}"
    echo "$SEARCH_OUTPUT"
    exit 1
fi
echo ""

# Test 5: CLI Search (Python code)
echo -e "${BLUE}Test 5: CLI Search (Python query)${NC}"
SEARCH_OUTPUT=$(./qwelli search "python programming" --index tests/demo/testdata/ --top 1 2>&1)
if echo "$SEARCH_OUTPUT" | grep -q "python_code.py"; then
    echo -e "${GREEN}✓ Python search working - found python_code.py${NC}"
else
    echo -e "${RED}✗ Python search failed${NC}"
    echo "$SEARCH_OUTPUT"
    exit 1
fi
echo ""

# Test 6: CLI Search (Hybrid)
echo -e "${BLUE}Test 6: CLI Search (Hybrid Strategy)${NC}"
SEARCH_OUTPUT=$(./qwelli search "cookies recipe" --index tests/demo/testdata/ --strategy hybrid --top 1 2>&1)
if echo "$SEARCH_OUTPUT" | grep -q "recipe.txt"; then
    echo -e "${GREEN}✓ Hybrid search working - found recipe.txt${NC}"
else
    echo -e "${RED}✗ Hybrid search failed${NC}"
    echo "$SEARCH_OUTPUT"
    exit 1
fi
echo ""

# Test 7: API Search
echo -e "${BLUE}Test 7: API Search${NC}"
API_OUTPUT=$(curl -s "http://localhost:8080/api/search?q=chocolate+chip+cookies&index=/home/karim-daw/dev/qwelli/tests/demo/testdata&top=1")
# Just check that API returns valid JSON with results
if echo "$API_OUTPUT" | grep -q '"results"' && echo "$API_OUTPUT" | grep -q '"query"'; then
    echo -e "${GREEN}✓ API search working (returns valid JSON)${NC}"
else
    echo -e "${RED}✗ API search failed${NC}"
    echo "$API_OUTPUT"
    exit 1
fi
echo ""

# Test 8: API Indexes
echo -e "${BLUE}Test 8: API List Indexes${NC}"
API_OUTPUT=$(curl -s "http://localhost:8080/api/indexes")
if echo "$API_OUTPUT" | grep -q "testdata"; then
    echo -e "${GREEN}✓ API list indexes working${NC}"
else
    echo -e "${RED}✗ API list indexes failed${NC}"
    echo "$API_OUTPUT"
    exit 1
fi
echo ""

# Test 9: Web UI
echo -e "${BLUE}Test 9: Web UI${NC}"
WEB_OUTPUT=$(curl -s http://localhost:8080/)
if echo "$WEB_OUTPUT" | grep -q "Qwelli"; then
    echo -e "${GREEN}✓ Web UI accessible${NC}"
else
    echo -e "${RED}✗ Web UI failed${NC}"
    exit 1
fi
echo ""

# Test 10: Different Search Strategies
echo -e "${BLUE}Test 10: Keyword Search${NC}"
SEARCH_OUTPUT=$(./qwelli search "chocolate" --index tests/demo/testdata/ --strategy keyword --top 1 2>&1)
if echo "$SEARCH_OUTPUT" | grep -q "Result 1"; then
    echo -e "${GREEN}✓ Keyword search working${NC}"
else
    echo -e "${RED}✗ Keyword search failed${NC}"
    echo "$SEARCH_OUTPUT"
    exit 1
fi
echo ""

# Test 11: Reranking (Default Enabled)
echo -e "${BLUE}Test 11: Reranking (Default Enabled)${NC}"
SEARCH_OUTPUT=$(./qwelli search "hello" --index tests/demo/testdata/ --verbose --top 2 2>&1)
if echo "$SEARCH_OUTPUT" | grep -q "Reranking"; then
    echo -e "${GREEN}✓ Reranking enabled by default${NC}"
else
    echo -e "${RED}✗ Reranking not working${NC}"
    echo "$SEARCH_OUTPUT"
    exit 1
fi
echo ""

# Test 12: Reranking Disabled
echo -e "${BLUE}Test 12: Reranking Disabled (--no-rerank)${NC}"
SEARCH_OUTPUT=$(./qwelli search "hello" --index tests/demo/testdata/ --no-rerank --verbose 2>&1)
if echo "$SEARCH_OUTPUT" | grep -q "disabled"; then
    echo -e "${GREEN}✓ --no-rerank flag working${NC}"
else
    echo -e "${RED}✗ --no-rerank flag not working${NC}"
    echo "$SEARCH_OUTPUT"
    exit 1
fi
echo ""

# Test 13: Status Command Shows Reranker
echo -e "${BLUE}Test 13: Status Command${NC}"
STATUS_OUTPUT=$(./qwelli status --index tests/demo/testdata/ 2>&1)
if echo "$STATUS_OUTPUT" | grep -q "Reranker: enabled"; then
    echo -e "${GREEN}✓ Status command shows reranker status${NC}"
else
    echo -e "${RED}✗ Status command missing reranker info${NC}"
    echo "$STATUS_OUTPUT"
    exit 1
fi
echo ""

# Summary
echo -e "${BLUE}=== Test Summary ===${NC}"
echo -e "${GREEN}✅ All 13 tests passed!${NC}"
echo ""
echo "Tested components:"
echo "  ✓ API health endpoints"
echo "  ✓ PostgreSQL with pgvector"
echo "  ✓ CLI list command"
echo "  ✓ CLI semantic search"
echo "  ✓ CLI search with specific queries"
echo "  ✓ CLI hybrid search strategy"
echo "  ✓ CLI keyword search strategy"
echo "  ✓ API search endpoint"
echo "  ✓ API list indexes endpoint"
echo "  ✓ Web UI accessibility"
echo "  ✓ Reranking enabled by default"
echo "  ✓ Reranking disable flag"
echo "  ✓ Status command shows reranker"
echo ""
echo -e "${BLUE}System is fully operational! 🚀${NC}"
echo ""
echo "Next steps:"
echo "  • Test the interactive shell: ./qwelli shell"
echo "  • Access web UI: http://localhost:8080"
echo "  • Index your own data: ./qwelli index /path/to/your/folder"
