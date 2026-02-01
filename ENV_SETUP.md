# Environment Variables Setup Guide

Qwelli now automatically loads `.env` files! This guide explains how it works and how to configure it.

## Quick Start

### 1. Copy the Example
```bash
cd /path/to/qwelli
cp .env.example .env
```

### 2. Edit Your API Key
```bash
# Edit .env and change this line:
VOYAGE_API_KEY=your_actual_api_key_here
```

### 3. Use CLI Commands
```bash
# No export needed!
./qwelli list
./qwelli search "hello world" --index /path/to/folder
./qwelli shell
```

That's it! The CLI automatically loads `.env` files.

---

## How It Works

### Automatic Loading

When you run any `qwelli` command, it automatically:

1. **Checks current directory** for `.env` file
2. **Searches parent directories** (up to 5 levels)
3. **Loads variables silently** (no error if file not found)
4. **Uses environment** for any already-set variables

This means:
- ✅ No need to manually `export` variables
- ✅ Works from any subdirectory
- ✅ Environment variables override `.env` file
- ✅ No changes needed to your shell config

### Priority Order

Variables are loaded in this order (last wins):

1. `.env` file values
2. Existing environment variables
3. Command-line overrides (if supported)

Example:
```bash
# .env has: DATABASE_URL=postgresql://localhost/db1
# But you want to use a different database temporarily:
DATABASE_URL=postgresql://localhost/db2 ./qwelli search "query"
```

---

## Configuration

### Required Variables

These **must** be set (either in `.env` or environment):

```bash
DATABASE_URL=postgresql://user:pass@host:5432/dbname?sslmode=disable
VOYAGE_API_KEY=your_voyage_api_key
```

### Optional Variables

These have sensible defaults:

```bash
VOYAGE_MODEL=voyage-multimodal-3  # Default: voyage-multimodal-3
PORT=8080                          # Default: 8080
ENABLE_RERANKER=true               # Default: true
```

---

## Different Scenarios

### Scenario 1: Local Development with Docker Compose

**Setup:**
```bash
# .env file
POSTGRES_PASSWORD=qwelli
DATABASE_URL=postgresql://qwelli:qwelli@localhost:5432/qwelli?sslmode=disable
VOYAGE_API_KEY=your_key
```

**Usage:**
```bash
docker-compose up -d
./qwelli index ~/Documents/my-project
./qwelli search "my query"
```

### Scenario 2: Remote PostgreSQL Database

**Setup:**
```bash
# .env file
DATABASE_URL=postgresql://user:pass@my-db.postgres.database.azure.com:5432/qwelli?sslmode=require
VOYAGE_API_KEY=your_key
```

**Usage:**
```bash
# No docker-compose needed
./qwelli index ~/Documents/my-project
./qwelli search "my query"
```

### Scenario 3: Multiple Projects

**Setup:**
```bash
# Different .env file per project
project-1/.env
project-2/.env
project-3/.env
```

**Usage:**
```bash
cd project-1
./qwelli search "query"  # Uses project-1/.env

cd ../project-2
./qwelli search "query"  # Uses project-2/.env
```

### Scenario 4: CI/CD Pipeline

In CI/CD, you typically set environment variables directly:

```bash
# GitHub Actions, etc.
export DATABASE_URL=${{ secrets.DATABASE_URL }}
export VOYAGE_API_KEY=${{ secrets.VOYAGE_API_KEY }}
./qwelli index /data/folder
```

The `.env` file is not needed in this case.

---

## Troubleshooting

### Problem: "DATABASE_URL environment variable is required"

**Solution 1:** Check `.env` file exists and has correct variable
```bash
cat .env | grep DATABASE_URL
```

**Solution 2:** Check you're in the right directory
```bash
pwd  # Should be in qwelli directory or subdirectory
ls .env  # Should exist
```

**Solution 3:** Create the `.env` file
```bash
cp .env.example .env
nano .env  # Edit with your values
```

### Problem: Wrong database being used

**Check what's loaded:**
```bash
# Add debug output (temporary)
DATABASE_URL=test ./qwelli list
# If it works, .env is being ignored due to env var override
```

**Solution:** Unset environment variable to use `.env`
```bash
unset DATABASE_URL
./qwelli list
```

### Problem: Changes to .env not taking effect

**Reason:** Environment variables override `.env` file

**Solution:** Check your shell profile
```bash
# Check if variables are set in your profile
cat ~/.bashrc | grep DATABASE_URL
cat ~/.zshrc | grep VOYAGE_API_KEY

# If found, either:
# 1. Remove from profile, or
# 2. Update the value there
```

---

## Advanced: Multiple .env Files

You can use different .env files for different purposes:

### .env.local (git-ignored)
```bash
# Your local overrides
DATABASE_URL=postgresql://localhost/qwelli_dev
VOYAGE_API_KEY=my_dev_key
```

### .env.production
```bash
# Production config
DATABASE_URL=postgresql://prod-host/qwelli
```

**Usage:**
```bash
# Use specific env file
cp .env.production .env
./qwelli search "query"

# Or use environment override
env $(cat .env.production) ./qwelli search "query"
```

---

## Best Practices

### ✅ DO

- ✅ Use `.env` for local development
- ✅ Add `.env` to `.gitignore` (already done)
- ✅ Commit `.env.example` with dummy values
- ✅ Use environment variables in production
- ✅ Keep `.env` file in project root

### ❌ DON'T

- ❌ Commit `.env` with real secrets to git
- ❌ Share `.env` files with credentials
- ❌ Use `.env` in production (use env vars instead)
- ❌ Hardcode credentials in code

---

## Migration from Manual Export

If you were previously doing this:

```bash
# OLD WAY (still works, but not needed)
export DATABASE_URL="postgresql://..."
export VOYAGE_API_KEY="..."
./qwelli search "query"
```

You can now do this:

```bash
# NEW WAY (simpler)
./qwelli search "query"
# Automatically loads from .env!
```

Your old `.env` exports in `.bashrc`/`.zshrc` still work and will override the file.

---

## Docker Compose Integration

The `.env` file serves **two purposes**:

1. **Docker Compose** reads it for container environment
2. **CLI binary** reads it for commands

This means one `.env` file works for both!

**Example .env for both:**
```bash
# Docker Compose uses these
POSTGRES_PASSWORD=qwelli
POSTGRES_DB=qwelli
POSTGRES_USER=qwelli

# CLI uses these
DATABASE_URL=postgresql://qwelli:qwelli@localhost:5432/qwelli?sslmode=disable
VOYAGE_API_KEY=your_key

# Both use these
PORT=8080
VOYAGE_MODEL=voyage-multimodal-3
```

---

## Summary

**Before (manual):**
```bash
export DATABASE_URL="..."
export VOYAGE_API_KEY="..."
./qwelli search "query"
```

**After (automatic):**
```bash
# Just have a .env file
./qwelli search "query"
```

**Much simpler!** 🎉
