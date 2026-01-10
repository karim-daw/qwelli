@echo off
setlocal enabledelayedexpansion

echo Building Qwelli with Web UI
echo.

REM Step 1: Build React frontend
echo Building React frontend...
cd web

if not exist "node_modules" (
    echo Installing frontend dependencies...
    call npm install
    if errorlevel 1 (
        echo Failed to install frontend dependencies
        exit /b 1
    )
)

echo Building frontend assets...
call npm run build
if errorlevel 1 (
    echo Failed to build frontend
    exit /b 1
)

cd ..

REM Step 2: Ensure the embed path exists
if not exist "internal\server\web\dist" (
    echo Creating embed directory...
    mkdir internal\server\web
)

REM Copy dist to embed location
echo Copying built assets to embed location...
if exist "internal\server\web\dist" rmdir /s /q internal\server\web\dist
xcopy /E /I /Y web\dist internal\server\web\dist >nul

REM Step 3: Build Go binary (using existing build script logic)
echo Building Go binary...

REM Check if MSYS2 is installed
if not exist "C:\msys64\usr\bin\pacman.exe" (
    echo MSYS2 not found. Please run build-windows.bat first.
    exit /b 1
)

REM Check if gcc is installed
gcc --version >nul 2>&1
if errorlevel 1 (
    if exist "C:\msys64\ucrt64\bin\gcc.exe" (
        set "PATH=C:\msys64\ucrt64\bin;!PATH!"
    ) else if exist "C:\msys64\mingw64\bin\gcc.exe" (
        set "PATH=C:\msys64\mingw64\bin;!PATH!"
    ) else (
        echo gcc not found. Please run build-windows.bat first.
        exit /b 1
    )
)

REM Check for DuckDB libraries
set "LIB_DIR=%~dp0build\libs"
if not exist "!LIB_DIR!\duckdb.dll" (
    echo DuckDB libraries not found. Please run download-duckdb.bat first.
    exit /b 1
)

set "PATH=!LIB_DIR!;!PATH!"
set "CGO_CFLAGS=-I!LIB_DIR!"
set "CGO_LDFLAGS=-L!LIB_DIR! -lduckdb"
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=1

go build -o qwelli.exe ./cmd/qwelli
if errorlevel 1 (
    echo Build failed
    exit /b 1
)

if not exist dist mkdir dist
copy qwelli.exe dist\ >nul
copy "!LIB_DIR!\duckdb.dll" dist\ >nul

echo.
echo Build complete!
echo    Binary: dist\qwelli.exe
echo.
echo To start the web UI, run:
echo    dist\qwelli.exe serve
