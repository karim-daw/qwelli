@echo off
setlocal enabledelayedexpansion

REM Check if MSYS2 is installed
if not exist "C:\msys64\usr\bin\pacman.exe" (
    REM MSYS2 not installed - download and install
    echo MSYS2 not found. Downloading installer...
    echo.
    set "MSYS2_INSTALLER=%TEMP%\msys2-x86_64-latest.exe"
    powershell -Command "Invoke-WebRequest -Uri 'https://github.com/msys2/msys2-installer/releases/latest/download/msys2-x86_64-latest.exe' -OutFile '%MSYS2_INSTALLER%'"
    if exist "%MSYS2_INSTALLER%" (
        echo.
        echo Starting MSYS2 installer. Please complete the installation, then run this script again.
        echo.
        start /wait "" "%MSYS2_INSTALLER%"
        echo.
        if not exist "C:\msys64\usr\bin\pacman.exe" (
            echo ❌ MSYS2 installation may have failed. Please install manually from https://www.msys2.org/
            exit /b 1
        )
        echo ✅ MSYS2 installed successfully
    ) else (
        echo ❌ Failed to download MSYS2 installer
        echo Please download and install MSYS2 manually from https://www.msys2.org/
        exit /b 1
    )
) else (
    echo ✅ MSYS2 already installed
)

REM Check if gcc is installed
gcc --version >nul 2>&1
if errorlevel 1 (
    REM gcc not in PATH, check MSYS2 locations
    if exist "C:\msys64\ucrt64\bin\gcc.exe" (
        set "PATH=C:\msys64\ucrt64\bin;!PATH!"
        echo ✅ Found gcc at C:\msys64\ucrt64\bin (added to PATH)
    ) else if exist "C:\msys64\mingw64\bin\gcc.exe" (
        set "PATH=C:\msys64\mingw64\bin;!PATH!"
        echo ✅ Found gcc at C:\msys64\mingw64\bin (added to PATH)
    ) else (
        REM gcc not found - install it
        echo gcc not found. Installing via MSYS2...
        echo.
        C:\msys64\usr\bin\bash.exe -lc "pacman -S --noconfirm mingw-w64-ucrt-x86_64-gcc"
        if not errorlevel 1 (
            set "PATH=C:\msys64\ucrt64\bin;!PATH!"
            echo ✅ gcc installed successfully
        ) else (
            echo ❌ Failed to install gcc. Please run manually:
            echo    Open MSYS2 shell and run: pacman -S mingw-w64-ucrt-x86_64-gcc
            exit /b 1
        )
    )
) else (
    echo ✅ gcc already available
)

REM Check for go
go version >nul 2>&1
if errorlevel 1 (
    echo ❌ Error: go not found in PATH
    exit /b 1
)

REM Check if DuckDB libraries exist
set "LIB_DIR=%~dp0build\libs"
if not exist "!LIB_DIR!\duckdb.dll" (
    echo DuckDB libraries not found in build\libs\
    echo.
    echo Please run download-duckdb.bat first, or
    echo download manually from: https://github.com/duckdb/duckdb/releases
    echo Look for: libduckdb-windows-amd64.zip
    echo.
    set /p DOWNLOAD="Download DuckDB libraries now? (Y/n): "
    if /i not "!DOWNLOAD!"=="n" (
        call download-duckdb.bat
        if errorlevel 1 (
            echo ❌ Failed to download DuckDB libraries
            exit /b 1
        )
    ) else (
        echo Please download DuckDB libraries and try again.
        exit /b 1
    )
)

set "PATH=!LIB_DIR!;!PATH!"
set "CGO_CFLAGS=-I!LIB_DIR!"
set "CGO_LDFLAGS=-L!LIB_DIR! -lduckdb"
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=1

go build -o qwelli.exe ./cmd/qwelli
if errorlevel 1 (
    echo ❌ Build failed
    exit /b 1
)

if not exist dist mkdir dist
copy qwelli.exe dist\ >nul
copy "!LIB_DIR!\duckdb.dll" dist\ >nul

echo ✅ Build complete: dist\qwelli.exe and dist\duckdb.dll

