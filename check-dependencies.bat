@echo off
echo Checking qwelli.exe and duckdb.dll dependencies...
echo.

if not exist "dist\qwelli.exe" (
    echo ❌ qwelli.exe not found in dist folder
    exit /b 1
)

if not exist "dist\duckdb.dll" (
    echo ❌ duckdb.dll not found in dist folder
    exit /b 1
)

echo ✅ Both files found in dist folder
echo.
echo To distribute:
echo   1. Zip the entire dist folder
echo   2. Users should extract and run qwelli.exe from the same folder as duckdb.dll
echo.
echo Note: Users may need Visual C++ Redistributable if not already installed.
echo Download: https://aka.ms/vs/17/release/vc_redist.x64.exe

