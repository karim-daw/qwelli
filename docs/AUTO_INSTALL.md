# Automatic Ollama Installation

Qwelli includes a sophisticated auto-installation system that detects your operating system and automatically installs Ollama if it's not found.

## Supported Platforms

The auto-installer supports:

- ✅ **Linux** (Ubuntu, Debian, Fedora, etc.)
- ✅ **WSL** (Windows Subsystem for Linux) - Detected automatically
- ✅ **macOS** (Intel and Apple Silicon)
- ✅ **Windows** (64-bit)

## Installation Locations

### Binary (Ollama Executable)

**Linux/WSL:**
- Binary: `/usr/local/bin/ollama` (or `/usr/bin/ollama`)
- Installation directory: `/usr/local` or `/usr`
- Service files: `/etc/systemd/system/ollama.service`
- Service user home: `/usr/share/ollama`

**macOS:**
- With Homebrew: `/opt/homebrew/bin/ollama` (Apple Silicon) or `/usr/local/bin/ollama` (Intel)
- Standalone: `/Applications/Ollama.app`

**Windows:**
- Binary: `C:\Program Files\Ollama\ollama.exe` (typically)
- Added to system PATH during installation

### Models Storage

**Linux/WSL:**
- User installation: `~/.ollama/models`
- System service: `/usr/share/ollama/.ollama/models`

**macOS:**
- `~/.ollama/models`

**Windows:**
- `C:\Users\%username%\.ollama\models`

### Model Directory Structure

```
.ollama/models/
├── blobs/          # Actual model data (content-addressed chunks)
└── manifests/      # Model manifests and metadata
```

### Customizing Storage Location

You can change where models are stored by setting the `OLLAMA_MODELS` environment variable:

```bash
# Linux/macOS
export OLLAMA_MODELS=/path/to/your/models

# Windows
set OLLAMA_MODELS=C:\path\to\your\models
```

## How It Works

### 1. OS Detection

The system automatically detects:
- Operating system (Linux, macOS, Windows)
- CPU architecture (amd64, arm64)
- WSL environment (if running in Windows Subsystem for Linux)

Example output:
```
🖥️  System: WSL (Windows Subsystem for Linux) (linux/arm64)
```

### 2. Installation Check

Checks if the `ollama` binary is available in the system PATH.

### 3. Automatic Installation

If Ollama is not found, the installer:

#### Linux / WSL
1. Downloads the official install script from `https://ollama.ai/install.sh`
2. Saves it to a temporary file
3. Makes it executable
4. Runs it (may prompt for sudo password)
5. Verifies installation

Console output:
```
⚠️  Ollama is not installed
🔧 Attempting automatic installation...
🔍 Detected OS: WSL (Windows Subsystem for Linux) (linux/arm64)
📦 Ollama not found. Starting automatic installation...
ℹ️  Running in WSL - using Linux installation method
📥 Downloading Ollama installation script...
🚀 Running Ollama installer...
   This may require sudo privileges and could take a few minutes...
   [installer] >>> Installing ollama to /usr/local
   [installer] >>> Downloading Linux arm64 bundle
   [installer] >>> Creating ollama user...
   [installer] >>> Adding current user to ollama group...
   [installer] >>> Creating ollama systemd service...
✅ Ollama installed successfully!
```

#### macOS
1. Checks if Homebrew is installed
   - If yes: Runs `brew install ollama`
   - If no: Downloads the installer package

Console output (with Homebrew):
```
⚠️  Ollama is not installed
🔧 Attempting automatic installation...
🍺 Homebrew detected. Installing Ollama via brew...
   [brew] ==> Downloading ollama...
   [brew] ==> Installing ollama...
✅ Ollama installed successfully via Homebrew!
```

Console output (without Homebrew):
```
⚠️  Ollama is not installed
🔧 Attempting automatic installation...
📥 Downloading Ollama installer for macOS...
   Detected Apple Silicon (ARM64)
📥 Downloading from https://ollama.ai/download/Ollama-darwin.zip
📦 Please complete the installation manually:
   1. Open Finder and navigate to: /tmp/ollama-install-xxxxx
   2. Extract Ollama.zip
   3. Drag Ollama.app to your Applications folder
   4. Open Ollama.app from Applications

⏳ Waiting for installation to complete...
   Press Enter once you've completed the installation...
```

#### Windows
1. Downloads `OllamaSetup.exe`
2. Runs the installer
3. Waits for user to complete installation wizard

Console output:
```
⚠️  Ollama is not installed
🔧 Attempting automatic installation...
🪟 Windows detected
📥 Downloading Ollama installer...
   From: https://ollama.ai/download/OllamaSetup.exe
🚀 Running installer...
   Please follow the installation wizard
⏳ Waiting for installation to complete...
   Press Enter once the installation wizard is finished...
```

### 4. Verification

After installation, the system verifies that `ollama` is accessible in the PATH.

## Console Output Guide

### Icons and Their Meanings

| Icon | Meaning |
|------|---------|
| 🖥️ | System information |
| ⚠️ | Warning or notice |
| 🔧 | Installation in progress |
| 🔍 | Detection or search |
| 📦 | Package/Installation |
| 📥 | Downloading |
| 🚀 | Starting process |
| ✅ | Success |
| ⏳ | Waiting |
| 🍺 | Homebrew (macOS) |
| 🪟 | Windows |
| ℹ️ | Information |
| 📊 | Progress update |
| 🛑 | Stopping |
| 🔢 | Number/Dimension |

### Typical Installation Flow

```
🖥️  System: Linux (linux/amd64)
⚠️  Ollama is not installed
🔧 Attempting automatic installation...
🔍 Detected OS: Linux (linux/amd64)
📦 Ollama not found. Starting automatic installation...
📥 Downloading Ollama installation script...
🚀 Running Ollama installer...
   This may require sudo privileges and could take a few minutes...
   [installer] >>> Installing ollama to /usr/local
   [installer] >>> Downloading Linux amd64 bundle
   [installer] >>> Creating ollama user...
   [installer] >>> Adding current user to ollama group...
   [installer] >>> Creating ollama systemd service...
✅ Ollama installed successfully!
✅ Ollama is installed
🚀 Starting Ollama server...
✅ Ollama server started (PID: 12345)
⏳ Waiting for Ollama server to be ready...
✅ Ollama server is ready
🔍 Checking for model 'nomic-embed-text'...
📥 Model 'nomic-embed-text' not found, downloading...
   This may take a few minutes depending on your connection speed...
   📊 pulling manifest
   📊 pulling 970aa74c0a90... 100%
   📊 pulling c71d239df917... 100%
   📊 verifying sha256 digest
   📊 writing manifest
   📊 success
✅ Model 'nomic-embed-text' downloaded successfully
🔢 Detecting embedding dimension...
✅ Ollama manager initialized
   Model: nomic-embed-text
   Dimension: 768
   Host: http://localhost:11434
```

## Implementation Details

### OS Detection (`internal/indexer/installer.go`)

```go
type OSInfo struct {
    OS           string // "linux", "darwin", "windows"
    Arch         string // "amd64", "arm64", etc.
    IsWSL        bool   // true if running in WSL
    DisplayName  string // Human-readable name
    InstallCmd   string // Installation command hint
}

func DetectOS() *OSInfo
```

### Installation Functions

- `IsOllamaInstalled()` - Checks if Ollama is in PATH
- `InstallOllama()` - Main installation dispatcher
- `installOllamaLinux()` - Linux/WSL installation
- `installOllamaMacOS()` - macOS installation
- `installOllamaWindows()` - Windows installation

### Integration

The auto-installer is integrated into `NewOllamaManager()` in `internal/indexer/ollama.go`:

```go
func NewOllamaManager(host, model string, managedServer bool) (*OllamaManager, error) {
    // Print OS information
    osInfo := DetectOS()
    log.Printf("🖥️  System: %s (%s/%s)", osInfo.DisplayName, osInfo.OS, osInfo.Arch)

    // Check if Ollama is installed, install if not
    if !IsOllamaInstalled() {
        log.Println("⚠️  Ollama is not installed")
        log.Println("🔧 Attempting automatic installation...")
        
        if err := InstallOllama(); err != nil {
            return nil, fmt.Errorf("failed to install Ollama: %w\n\nPlease install manually using: %s", err, osInfo.InstallCmd)
        }
        
        log.Println("✅ Ollama installation complete!")
    } else {
        log.Println("✅ Ollama is installed")
    }
    
    // ... continue with initialization
}
```

## Troubleshooting

### Permission Denied (Linux/WSL)

If you see permission errors during installation:
```
   [installer] Error: Permission denied
```

Make sure to enter your sudo password when prompted, or run with sudo:
```bash
sudo go run your-app.go
```

### Installation Failed

If automatic installation fails, you'll see:
```
failed to install Ollama: [error details]

Please install manually using: [platform-specific command]
```

Follow the manual installation instructions for your platform.

### Ollama Not Found After Installation

On Windows, you may need to restart your terminal or add Ollama to your PATH manually.

On Linux/macOS, you may need to start a new shell session:
```bash
source ~/.bashrc  # or ~/.zshrc
```

### WSL-Specific Issues

If running in WSL and installation fails, you can:
1. Install Ollama in Windows (recommended)
2. Connect to the Windows Ollama server from WSL
3. Manually install in WSL using the Linux method

## Disabling Auto-Install

If you want to disable auto-install and handle Ollama installation yourself, you can check before creating the manager:

```go
import "github.com/karim-daw/qwelli/internal/indexer"

if !indexer.IsOllamaInstalled() {
    log.Fatal("Ollama is not installed. Please install it first.")
}

// This will skip auto-install since Ollama is already present
embedder, err := indexer.NewEmbedder("nomic-embed-text", "", 0)
```

## Security Considerations

The auto-installer:
- Downloads official installation scripts/binaries from ollama.ai
- Verifies HTTP status codes (200 OK)
- Does NOT verify cryptographic signatures (planned for future)
- May require sudo/admin privileges on Linux
- Prompts user for interaction on macOS and Windows

For production environments, consider:
- Pre-installing Ollama in your deployment pipeline
- Using containerized deployments with Ollama pre-installed
- Manually verifying installation scripts before running
