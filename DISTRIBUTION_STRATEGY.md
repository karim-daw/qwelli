# Distribution Strategy for All Platforms

## Supported Platforms

### Tier 1: Automated via GitHub Actions
These build automatically when you push a tag:

- ✅ **Linux AMD64** - Most common Linux (desktop/server)
- ✅ **macOS Intel (AMD64)** - Intel Macs
- ✅ **macOS Apple Silicon (ARM64)** - M1/M2/M3 Macs
- ✅ **Windows AMD64** - Standard Windows PCs

### Tier 2: Manual Build Required
These require manual builds due to GitHub Actions limitations:

- 🔧 **Linux ARM64** - Raspberry Pi, AWS Graviton, etc.
- 🔧 **Windows ARM64** - Surface Pro X, Windows on ARM

## Distribution Workflows

### Workflow 1: Automated Release (Most Users)

This covers 95%+ of users:

```bash
# 1. Tag and push
git tag v0.1.0
git push origin v0.1.0

# 2. GitHub Actions automatically builds 4 platforms
# 3. Release is created with binaries
```

**Users get:**
- qwelli-linux-amd64
- qwelli-macos-amd64
- qwelli-macos-arm64
- qwelli-windows-amd64.exe

### Workflow 2: Full Release (All Platforms)

For complete platform coverage:

```bash
# Step 1: Let GitHub Actions create the release (4 platforms)
git tag v0.1.0
git push origin v0.1.0

# Step 2: Wait for GitHub Actions to complete

# Step 3: Build ARM64 variants manually
# On Linux ARM64 machine:
./upload-arm64.sh v0.1.0

# On Windows ARM64 machine (if you have one):
./upload-arm64.sh v0.1.0
```

**Users get all 6 platforms:**
- qwelli-linux-amd64
- qwelli-linux-arm64 ⭐
- qwelli-macos-amd64
- qwelli-macos-arm64
- qwelli-windows-amd64.exe
- qwelli-windows-arm64.exe ⭐

### Workflow 3: Self-Hosted Runners (Advanced)

For fully automated builds of all platforms:

1. **Set up self-hosted runners:**
   - Linux ARM64 runner (Raspberry Pi, cloud VM, etc.)
   - Windows ARM64 runner (if needed)

2. **Use `.github/workflows/release-full.yml`:**
   ```bash
   # Uncomment the self-hosted runner sections
   # in .github/workflows/release-full.yml
   ```

3. **Push tags as normal:**
   ```bash
   git tag v0.1.0
   git push origin v0.1.0
   ```

All 6 platforms build automatically!

## Decision Guide

### For Most Projects (Recommended)
**Use Workflow 1** - GitHub Actions only

- Covers 95%+ of users
- Fully automated
- Zero maintenance
- ARM64 users can build from source

### For Wide Distribution
**Use Workflow 2** - GitHub Actions + Manual ARM64

- Covers all users
- Minimal manual work (2-5 minutes per release)
- No infrastructure needed
- Just need access to ARM64 machines

### For High-Volume Projects
**Use Workflow 3** - Self-hosted runners

- Fully automated
- Requires maintaining runners
- Best for frequent releases
- More complex setup

## ARM64 Build Instructions

### Linux ARM64

On an ARM64 Linux machine:

```bash
# Clone repo
git clone https://github.com/YOUR_USERNAME/qwelli.git
cd qwelli

# Checkout the release tag
git checkout v0.1.0

# Build
./build-local.sh

# Upload to GitHub release
chmod +x upload-arm64.sh
./upload-arm64.sh v0.1.0
```

### Windows ARM64

On a Windows ARM64 machine:

```powershell
# Clone repo
git clone https://github.com/YOUR_USERNAME/qwelli.git
cd qwelli

# Checkout the release tag
git checkout v0.1.0

# Build
go build -ldflags="-s -w" -o build/qwelli-windows-arm64.exe ./cmd/qwelli

# Upload using gh CLI
gh release upload v0.1.0 build/qwelli-windows-arm64.exe --clobber
```

## Installation Instructions for Users

### Auto-detect Platform Script

Create an install script that detects the user's platform:

```bash
curl -fsSL https://raw.githubusercontent.com/YOUR_USERNAME/qwelli/main/install.sh | bash
```

This would detect their platform and download the appropriate binary.

### Manual Download

Users go to releases page:
```
https://github.com/YOUR_USERNAME/qwelli/releases/latest
```

And download the binary for their platform.

## Platform Priorities

Based on market share:

1. **Linux AMD64** (servers, most desktops) - ~60% of Linux users
2. **Windows AMD64** (most PCs) - ~95% of Windows users
3. **macOS Apple Silicon** (newer Macs) - ~70% of Mac users
4. **macOS Intel** (older Macs) - ~30% of Mac users
5. **Linux ARM64** (Raspberry Pi, servers) - ~5% of Linux users
6. **Windows ARM64** (Surface, etc.) - ~1% of Windows users

**Recommendation:** Start with Workflow 1 (automated 4 platforms), add ARM64 builds only if users request them.

## Cost Analysis

| Workflow | Setup Time | Time per Release | Infrastructure Cost |
|----------|------------|------------------|---------------------|
| Workflow 1 | 5 min | 0 min (automatic) | $0 |
| Workflow 2 | 10 min | 5 min (manual ARM builds) | $0 |
| Workflow 3 | 2-4 hours | 0 min (automatic) | $5-20/month |

## Summary

**Start simple (Workflow 1):**
- Push tags
- Let GitHub Actions do the work
- 4 platforms automatically
- Add ARM64 later if needed

**Your current setup is ready for Workflow 1!** 🚀
