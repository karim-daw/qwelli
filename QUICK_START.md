# Qwelli Quick Start Guide

Get up and running with Qwelli in 3 minutes.

## Setup (One-Time)

### 1. Copy Environment File
```bash
cd /path/to/qwelli
cp .env.example .env
```

### 2. Add Your Voyage API Key
```bash
# Edit .env and change this line:
nano .env

# Change:
VOYAGE_API_KEY=your_voyage_api_key_here
# To:
VOYAGE_API_KEY=your_actual_key_from_voyageai_com
```

### 3. Start Docker Compose
```bash
docker-compose up -d
```

**That's it!** You're ready to use Qwelli.

---

## Basic Usage

### Index Files
```bash
# No need to export environment variables!
./qwelli index ~/Documents/my-project
```

### Search
```bash
./qwelli search "machine learning" --index ~/Documents/my-project
```

### List All Indexes
```bash
./qwelli list
```

### Interactive Shell
```bash
./qwelli shell
```

### Web UI
```bash
# Already running with docker-compose
# Open: http://localhost:8080
```

---

## How It Works

**Automatic .env Loading:**
- CLI automatically finds and loads `.env` file
- Works from any subdirectory (searches up to 5 levels)
- No manual `export` commands needed
- Just run `./qwelli <command>` and it works!

**What happens:**
1. You run `./qwelli search ...`
2. CLI looks for `.env` in current directory
3. If not found, checks parent directories
4. Loads variables automatically
5. Runs your command

---

## Common Commands

```bash
# Index a folder
./qwelli index /path/to/folder

# Search with different strategies
./qwelli search "query"  # semantic (default)
./qwelli search "query" --strategy hybrid
./qwelli search "query" --strategy keyword

# Limit results
./qwelli search "query" --top 10

# Interactive mode
./qwelli shell

# Web UI
# Just open http://localhost:8080 in browser
```

---

## Verify Setup

Run the automated test suite:
```bash
./test-all.sh
```

Should show:
```
✅ All 10 tests passed!
```

---

## Troubleshooting

### "DATABASE_URL environment variable is required"

**Check .env file exists:**
```bash
ls .env
cat .env | grep DATABASE_URL
```

**If missing, create it:**
```bash
cp .env.example .env
nano .env  # Add your VOYAGE_API_KEY
```

### Docker containers not running

```bash
docker-compose ps
# If not running:
docker-compose up -d
```

### Port already in use

```bash
# Change port in .env:
echo "APP_PORT=3000" >> .env
docker-compose down
docker-compose up -d
```

---

## Next Steps

1. **Index your data:**
   ```bash
   ./qwelli index ~/Documents/important-project
   ```

2. **Try different searches:**
   ```bash
   ./qwelli search "API documentation"
   ./qwelli search "authentication code" --strategy hybrid
   ```

3. **Use the web UI:**
   - Open http://localhost:8080
   - Visual interface for searching
   - Browse indexed files

4. **Read full documentation:**
   - `TESTING.md` - Comprehensive testing guide
   - `DEPLOYMENT.md` - Production deployment
   - `ENV_SETUP.md` - Environment variable details

---

## Environment File Structure

Your `.env` file should look like this:

```bash
# For docker-compose
POSTGRES_PASSWORD=qwelli

# For CLI commands (auto-loaded!)
DATABASE_URL=postgresql://qwelli:qwelli@localhost:5432/qwelli?sslmode=disable

# Your API key
VOYAGE_API_KEY=your_actual_key

# Optional settings
VOYAGE_MODEL=voyage-multimodal-3
PORT=8080
ENABLE_RERANKER=true
```

**Key points:**
- `POSTGRES_PASSWORD` - Used by docker-compose to set up PostgreSQL
- `DATABASE_URL` - Used by CLI to connect to PostgreSQL
- `VOYAGE_API_KEY` - Required for embedding generation
- Everything else has sensible defaults

---

## Production

For production with managed PostgreSQL:

```bash
# Update DATABASE_URL in .env:
DATABASE_URL=postgresql://user:pass@your-managed-db.azure.com:5432/qwelli?sslmode=require

# Deploy with production config:
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d qwelli
```

See `DEPLOYMENT.md` for complete production guide.

---

## Summary

**Before:**
```bash
export DATABASE_URL="..."
export VOYAGE_API_KEY="..."
./qwelli search "query"
```

**Now:**
```bash
# Just have a .env file
./qwelli search "query"
```

**Much simpler!** 🎉
