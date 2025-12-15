# Qwelli Windows Installer

This directory contains the WiX installer configuration for creating an MSI installer for Qwelli.

## Prerequisites

1. **WiX Toolset** - Download and install from https://wixtoolset.org/releases/
   - Install the latest stable version
   - Make sure `candle.exe` and `light.exe` are in your PATH

2. **Built Application** - Run `build-windows.bat` first to create `dist/qwelli.exe` and `dist/duckdb.dll`

## Building the Installer

1. Navigate to the `installer` directory:
   ```cmd
   cd installer
   ```

2. Run the build script:
   ```cmd
   build-installer.bat
   ```

3. The MSI installer will be created as `qwelli-installer.msi`

## What the Installer Does

- Installs `qwelli.exe` and `duckdb.dll` to `C:\Program Files\Qwelli\`
- Creates a Start Menu shortcut
- Optionally adds Qwelli to system PATH (user can choose during installation)
- Provides uninstaller via Windows Add/Remove Programs

## Customization

Edit `qwelli.wxs` to customize:
- Product name and version
- Installation directory
- Features (Start Menu, PATH, etc.)
- Upgrade code (generate a GUID for production)

## Testing

To test the installer:
```cmd
msiexec /i qwelli-installer.msi
```

Or simply double-click `qwelli-installer.msi`

## Uninstalling

Users can uninstall via:
- Windows Settings > Apps > Qwelli > Uninstall
- Control Panel > Programs and Features > Qwelli > Uninstall

