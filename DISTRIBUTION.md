# Qwelli Distribution Guide

## Building Executables for Distribution

### Quick Build

```bash
# Build for all platforms
./build.sh

# Or build for specific platform
GOOS=windows GOARCH=amd64 go build -o qwelli.exe ./cmd/qwelli
```

### Output

Executables will be in `build/` directory:
- `qwelli-windows-amd64.exe` - Windows 64-bit
- `qwelli-macos-amd64` - macOS Intel
- `qwelli-macos-arm64` - macOS Apple Silicon
- `qwelli-linux-amd64` - Linux 64-bit
- `qwelli-linux-arm64` - Linux ARM

## Making It "Safe" for Distribution

### 1. Code Signing (Recommended for Production)

#### Windows
```bash
# Requires Windows code signing certificate
signtool sign /f certificate.pfx /p password /t http://timestamp.digicert.com qwelli.exe
```

#### macOS
```bash
# Requires Apple Developer account
codesign -s "Developer ID Application: Your Name" qwelli-macos-amd64
codesign -s "Developer ID Application: Your Name" qwelli-macos-arm64

# Notarize for Gatekeeper
xcrun notarytool submit qwelli-macos-amd64.zip --apple-id your@email.com --wait
xcrun stapler staple qwelli-macos-amd64
```

### 2. Security Best Practices

**Current Implementation:**
- ✅ API keys stored in `~/.qwelli/config.yaml` with 0600 permissions (user-only read/write)
- ✅ All data stays local (privacy-first)
- ✅ No telemetry or phone-home
- ✅ Open source (customers can audit)

**Additional Recommendations:**
- Add checksum file (SHA256) for download verification
- Use HTTPS for all downloads
- Provide GPG signatures for Linux users

### 3. Create Release Package

```bash
cd build

# Windows
zip qwelli-windows-amd64-v0.1.0.zip qwelli-windows-amd64.exe

# macOS
tar -czf qwelli-macos-amd64-v0.1.0.tar.gz qwelli-macos-amd64
tar -czf qwelli-macos-arm64-v0.1.0.tar.gz qwelli-macos-arm64

# Linux
tar -czf qwelli-linux-amd64-v0.1.0.tar.gz qwelli-linux-amd64

# Create checksums
sha256sum * > SHA256SUMS.txt
```

### 4. Distribution Methods

#### Option A: GitHub Releases (Recommended)
1. Create a GitHub release
2. Upload all binaries + SHA256SUMS.txt
3. Customers download directly from GitHub
4. Free, trusted, with version history

#### Option B: Your Website
1. Host binaries on your server
2. Provide download page with checksums
3. Include installation instructions

#### Option C: Package Managers

**Homebrew (macOS/Linux):**
```ruby
# Create a homebrew formula
class Qwelli < Formula
  desc "Semantic search for local files"
  homepage "https://github.com/yourusername/qwelli"
  url "https://github.com/yourusername/qwelli/archive/v0.1.0.tar.gz"
  sha256 "..."

  def install
    system "go", "build", "-o", bin/"qwelli", "./cmd/qwelli"
  end
end
```

**Chocolatey (Windows):**
```powershell
# Create chocolatey package
choco pack
choco push
```

**Snap (Linux):**
```yaml
# snapcraft.yaml
name: qwelli
version: '0.1.0'
summary: Semantic search for local files
apps:
  qwelli:
    command: qwelli
```

## Installation Instructions for Customers

### Windows

1. Download `qwelli-windows-amd64.exe`
2. Rename to `qwelli.exe`
3. Move to a folder in your PATH (e.g., `C:\Program Files\Qwelli`)
4. Or add the folder to your PATH
5. Open Command Prompt and run `qwelli init`

### macOS

```bash
# Download and extract
curl -LO https://github.com/yourusername/qwelli/releases/download/v0.1.0/qwelli-macos-arm64.tar.gz
tar -xzf qwelli-macos-arm64.tar.gz

# Make executable
chmod +x qwelli-macos-arm64

# Move to PATH
sudo mv qwelli-macos-arm64 /usr/local/bin/qwelli

# Run
qwelli init
```

### Linux

```bash
# Download and extract
wget https://github.com/yourusername/qwelli/releases/download/v0.1.0/qwelli-linux-amd64.tar.gz
tar -xzf qwelli-linux-amd64.tar.gz

# Make executable
chmod +x qwelli-linux-amd64

# Move to PATH
sudo mv qwelli-linux-amd64 /usr/local/bin/qwelli

# Run
qwelli init
```

## Antivirus False Positives

Go binaries sometimes trigger false positives in antivirus software because:
- They're statically compiled (include all dependencies)
- Often unsigned (for free/open source tools)

**Solutions:**
1. **Code signing** (best solution - costs ~$300/year for Windows certificate)
2. Submit to antivirus vendors for whitelisting
3. Provide source code + build instructions for transparency
4. Use UPX compression (sometimes helps, sometimes makes it worse)

## Versioning

Use semantic versioning in `cmd/qwelli/main.go`:

```go
var version = "0.1.0"
```

Increment for releases:
- `0.1.0` → `0.1.1` (bug fixes)
- `0.1.0` → `0.2.0` (new features)
- `0.1.0` → `1.0.0` (major changes)

## Build Flags Explained

```bash
go build -ldflags="-s -w" -o qwelli ./cmd/qwelli
```

- `-s` - Strip debug symbols (smaller binary)
- `-w` - Strip DWARF debug info (smaller binary)
- Result: ~30-50% smaller executables

## Testing Before Release

```bash
# Build
./build.sh

# Test each binary
./build/qwelli-linux-amd64 --version
./build/qwelli-linux-amd64 --help

# Full test
./build/qwelli-linux-amd64 init
./build/qwelli-linux-amd64 index tests/demo/testdata
./build/qwelli-linux-amd64 search "hello" --index tests/demo/testdata
```

## Distribution Checklist

- [ ] Update version number in `cmd/qwelli/main.go`
- [ ] Run `./build.sh`
- [ ] Test each platform binary
- [ ] Create SHA256 checksums
- [ ] Write release notes
- [ ] Tag git release: `git tag v0.1.0`
- [ ] Create GitHub release
- [ ] Upload binaries + checksums
- [ ] Update documentation
- [ ] Announce release

## Auto-Update (Future Enhancement)

Consider adding auto-update functionality:
- Check GitHub releases API for new versions
- Download and replace binary
- Libraries: `github.com/rhysd/go-github-selfupdate`

## Licensing

Add a LICENSE file (e.g., MIT, Apache 2.0) so customers know it's safe to use commercially.

## Support

Provide customer support channels:
- GitHub Issues for bug reports
- Documentation site
- Email support
- Discord/Slack community
