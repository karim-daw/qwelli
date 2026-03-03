@echo off
setlocal enabledelayedexpansion

echo ============================================
echo   Qwelli Build
echo ============================================
echo.

REM Parse arguments
set "RELEASE=0"
set "DESKTOP=0"
for %%a in (%*) do (
    if "%%a"=="--release" set "RELEASE=1"
    if "%%a"=="--desktop" set "DESKTOP=1"
)

REM Get script directory and project root
set "SCRIPT_DIR=%~dp0"
set "ROOT=%SCRIPT_DIR%.."
cd /d "%ROOT%"

REM ============================================
REM Step 1: Check prerequisites
REM ============================================
echo [1/4] Checking prerequisites...

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

if "%DESKTOP%"=="1" (
    go list github.com/wailsapp/wails/v2 >nul 2>&1
    if errorlevel 1 (
        echo   Installing Wails dependency...
        go get github.com/wailsapp/wails/v2
        if errorlevel 1 (
            echo X Failed to fetch Wails dependency
            exit /b 1
        )
    )
    echo   + Wails found
)
echo.

REM ============================================
REM Step 2: Build React frontend
REM ============================================
echo [2/4] Building React frontend...

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
echo [3/4] Copying frontend to embed location...

if exist "internal\server\web\dist" rmdir /s /q "internal\server\web\dist"
if not exist "internal\server\web" mkdir "internal\server\web"
xcopy /E /I /Y /Q "web\dist" "internal\server\web\dist" >nul
echo   + Done
echo.

REM ============================================
REM Step 4: Build Go binary
REM ============================================
echo [4/4] Building Go binary...

if not exist "build" mkdir "build"

if "%DESKTOP%"=="1" (
    set "OUTPUT=build\qwelli-desktop.exe"
    set "TAGS=-tags desktop,production"
) else (
    set "OUTPUT=build\qwelli.exe"
    set "TAGS="
)

if "%RELEASE%"=="1" (
    echo   Mode: release
    go build %TAGS% -ldflags "-s -w -H windowsgui" -o "%OUTPUT%" ./cmd/qwelli
) else (
    echo   Mode: dev
    go build %TAGS% -o "%OUTPUT%" ./cmd/qwelli
)
if errorlevel 1 (
    echo X Build failed
    exit /b 1
)
echo   + %OUTPUT%
echo.

echo ============================================
echo   Build Complete!
echo ============================================
echo.
if "%DESKTOP%"=="1" (
    echo   Run: %OUTPUT%
) else (
    echo   Run: build\qwelli.exe serve
)
echo.

exit /b 0
