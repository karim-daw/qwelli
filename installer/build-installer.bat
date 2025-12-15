@echo off
setlocal enabledelayedexpansion

echo Building Qwelli MSI Installer...
echo.

REM Check if WiX is installed
where candle.exe >nul 2>&1
if errorlevel 1 (
    echo ❌ WiX Toolset not found in PATH
    echo.
    echo Please install WiX Toolset:
    echo   1. Download from: https://wixtoolset.org/releases/
    echo   2. Install WiX Toolset (candle.exe and light.exe)
    echo   3. Add WiX bin directory to PATH
    echo   4. Run this script again
    exit /b 1
)

echo ✅ WiX Toolset found
echo.

REM Check if dist files exist
if not exist "..\dist\qwelli.exe" (
    echo ❌ qwelli.exe not found in dist folder
    echo Please run build-windows.bat first
    exit /b 1
)

if not exist "..\dist\duckdb.dll" (
    echo ❌ duckdb.dll not found in dist folder
    echo Please run build-windows.bat first
    exit /b 1
)

echo ✅ Source files found
echo.

REM Compile WiX source
echo Compiling WiX source...
candle.exe qwelli.wxs -out qwelli.wixobj
if errorlevel 1 (
    echo ❌ Failed to compile WiX source
    exit /b 1
)

REM Link to create MSI
echo Linking MSI installer...
light.exe qwelli.wixobj -out qwelli-installer.msi
if errorlevel 1 (
    echo ❌ Failed to create MSI
    exit /b 1
)

echo.
echo ✅ Installer created: qwelli-installer.msi
echo.
echo To test the installer:
echo   msiexec /i qwelli-installer.msi
echo.
echo Or double-click qwelli-installer.msi to install

