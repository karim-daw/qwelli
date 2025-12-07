# Building Qwelli

## The Cross-Compilation Challenge

Qwelli uses **DuckDB** which relies on **CGo** (C bindings). This makes cross-compilation challenging because you need the C compiler and libraries for each target platform.

## Build Options

### Option 1: Local Build (Recommended for Development)

Build only for your current platform:

```bash
./build-local.sh
```

Or directly:

```bash
go build -o qwelli ./cmd/qwelli
```

### Option 2: Cross-Platform Build (Limited)

The `build.sh` script attempts to build for all platforms, but **will fail for non-native platforms** due to CGo requirements:

```bash
./build.sh
```

**Expected behavior:**
- ✅ Builds successfully for your current platform
- ❌ Fails for other platforms (Windows, macOS, other Linux architectures)

### Option 3: GitHub Actions (Recommended for Releases)

For production releases, use GitHub Actions which builds on native platforms:

1. **Push a tag:**
   ```bash
   git tag v0.1.0
   git push origin v0.1.0
   ```

2. GitHub Actions will automatically:
   - Build on actual Windows, macOS, and Linux runners
   - Create binaries for all platforms
   - Attach them to a GitHub Release

3. **Manual trigger** (without tag):
   ```bash
   # Go to GitHub Actions tab and manually run "Build Release" workflow
   ```

## Platform-Specific Notes

### Linux
- Native builds work out of the box
- Cross-compiling to ARM64 from AMD64 (or vice versa) requires cross-compilation toolchain

### macOS
- Can only build macOS binaries on macOS
- Use GitHub Actions or build on an actual Mac

### Windows
- Can only build Windows binaries on Windows
- Use GitHub Actions or build on an actual Windows machine

## Alternative: Docker Multi-Platform Builds

For advanced users, you can use Docker with cross-compilation toolchains:

```bash
# Example for Linux targets only
docker run --rm -v "$PWD":/workspace -w /workspace \
  golang:1.21 \
  go build -o qwelli-docker ./cmd/qwelli
```

However, this still requires CGo cross-compilation setup for non-native platforms.

## Troubleshooting

### "undefined: bindings.Type" errors
This is expected when trying to cross-compile with CGo. Use one of these solutions:
- Build on the native platform
- Use GitHub Actions
- Use the local build script for development

### Build succeeds but binary doesn't work
Make sure you're running the binary built for your platform:
- Linux: `qwelli-linux-amd64` or `qwelli-linux-arm64`
- macOS Intel: `qwelli-macos-amd64`
- macOS Apple Silicon: `qwelli-macos-arm64`
- Windows: `qwelli-windows-amd64.exe`

## Quick Reference

| What you need | Command |
|---------------|---------|
| Local development | `./build-local.sh` or `go build -o qwelli ./cmd/qwelli` |
| Test build | `go build ./cmd/qwelli` |
| Production release | Push a git tag, let GitHub Actions build |
| All platforms (will partially fail) | `./build.sh` |
