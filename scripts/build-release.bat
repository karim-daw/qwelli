@echo off
setlocal enabledelayedexpansion

echo ============================================
echo   Qwelli Windows Release Build
echo ============================================
echo.

REM Get script directory and project root
set "SCRIPT_DIR=%~dp0"
set "ROOT=%SCRIPT_DIR%.."
cd /d "%ROOT%"

REM ============================================
REM Step 1: Check prerequisites
REM ============================================
echo [1/5] Checking prerequisites...

go version >nul 2>&1
if errorlevel 1 (
    echo X Error: Go not found in PATH
    exit /b 1
)
echo   + Go found

node --version >nul 2>&1
if errorlevel 1 (
    echo X Error: Node.js not found in PATH
    exit /b 1
)
echo   + Node.js found
echo.

REM ============================================
REM Step 2: Build React frontend
REM ============================================
echo [2/5] Building React frontend...

cd "%ROOT%\web"

if not exist "node_modules" (
    echo   Installing dependencies...
    call npm install --silent
    if errorlevel 1 (
        echo X Failed to install frontend dependencies
        exit /b 1
    )
)

call npm run build --silent
if errorlevel 1 (
    echo X Failed to build frontend
    exit /b 1
)

cd "%ROOT%"
echo   + Frontend built
echo.

REM ============================================
REM Step 3: Copy frontend to embed location
REM ============================================
echo [3/5] Copying frontend to embed location...

if exist "internal\server\web\dist" rmdir /s /q "internal\server\web\dist"
if not exist "internal\server\web" mkdir "internal\server\web"
xcopy /E /I /Y /Q "web\dist" "internal\server\web\dist" >nul
echo   + Done
echo.

REM ============================================
REM Step 4: Build Go binary
REM ============================================
echo [4/5] Building Go binary...

if not exist "build" mkdir "build"

go build -ldflags "-s -w -H windowsgui" -o "build\qwelli.exe" ./cmd/qwelli
if errorlevel 1 (
    echo X Build failed
    exit /b 1
)
echo   + build\qwelli.exe
echo.

REM ============================================
REM Step 5: Create release package
REM ============================================
echo [5/5] Creating release package...

if not exist "dist" mkdir "dist"
copy "build\qwelli.exe" "dist\" >nul

set "ZIP_NAME=qwelli-windows-amd64.zip"
cd "dist"
powershell -Command "Compress-Archive -Path 'qwelli.exe' -DestinationPath '..\%ZIP_NAME%' -Force"
cd "%ROOT%"

echo   + dist\qwelli.exe
echo   + %ZIP_NAME%
echo.

echo ============================================
echo   Build Complete!
echo ============================================
echo.
echo   Run: dist\qwelli.exe serve
echo.

exit /b 0
