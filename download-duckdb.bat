@echo off
setlocal enabledelayedexpansion

echo Downloading DuckDB libraries for Windows...
echo.

REM Create build/libs directory if it doesn't exist
if not exist "build\libs" (
    echo Creating build\libs directory...
    mkdir "build\libs"
)

REM Check if files already exist
if exist "build\libs\duckdb.dll" (
    echo DuckDB files already exist in build\libs\
    echo.
    set /p OVERWRITE="Do you want to re-download? (y/N): "
    if /i not "!OVERWRITE!"=="y" (
        echo Skipping download.
        exit /b 0
    )
    echo.
)

REM Download DuckDB Windows library
echo Downloading DuckDB Windows library...
echo.

REM Use specific DuckDB version
set "DUCKDB_VERSION=v1.4.3"

set "DUCKDB_URL=https://github.com/duckdb/duckdb/releases/download/%DUCKDB_VERSION%/libduckdb-windows-amd64.zip"
set "ZIP_FILE=%TEMP%\libduckdb-windows-amd64.zip"

echo DuckDB version: %DUCKDB_VERSION%
echo Downloading from: %DUCKDB_URL%
echo.

powershell -Command "try { $ProgressPreference = 'SilentlyContinue'; Invoke-WebRequest -Uri '%DUCKDB_URL%' -OutFile '%ZIP_FILE%' -ErrorAction Stop; Write-Host 'Download complete' } catch { Write-Host 'Download failed:'; Write-Host $_.Exception.Message; exit 1 }"

if errorlevel 1 (
    echo.
    echo ❌ Failed to download DuckDB library
    echo.
    echo Please download manually from:
    echo https://github.com/duckdb/duckdb/releases
    echo.
    echo Look for: libduckdb-windows-amd64.zip
    echo Extract the .dll, .lib, .h, and .hpp files to build\libs\
    exit /b 1
)

REM Extract files
echo.
echo Extracting files to build\libs\...
echo.

powershell -Command "$ProgressPreference = 'SilentlyContinue'; Expand-Archive -Path '%ZIP_FILE%' -DestinationPath '%TEMP%\duckdb-extract' -Force"

if errorlevel 1 (
    echo ❌ Failed to extract zip file
    exit /b 1
)

REM Copy files to build/libs
if exist "%TEMP%\duckdb-extract\duckdb.dll" (
    copy "%TEMP%\duckdb-extract\duckdb.dll" "build\libs\" >nul
    echo ✅ Copied duckdb.dll
) else (
    echo ❌ duckdb.dll not found in archive
)

if exist "%TEMP%\duckdb-extract\duckdb.lib" (
    copy "%TEMP%\duckdb-extract\duckdb.lib" "build\libs\" >nul
    echo ✅ Copied duckdb.lib
) else (
    echo ❌ duckdb.lib not found in archive
)

if exist "%TEMP%\duckdb-extract\duckdb.h" (
    copy "%TEMP%\duckdb-extract\duckdb.h" "build\libs\" >nul
    echo ✅ Copied duckdb.h
) else (
    echo ❌ duckdb.h not found in archive
)

if exist "%TEMP%\duckdb-extract\duckdb.hpp" (
    copy "%TEMP%\duckdb-extract\duckdb.hpp" "build\libs\" >nul
    echo ✅ Copied duckdb.hpp
) else (
    echo ❌ duckdb.hpp not found in archive
)

REM Cleanup
powershell -Command "Remove-Item -Path '%TEMP%\duckdb-extract' -Recurse -Force -ErrorAction SilentlyContinue"
del "%ZIP_FILE%" >nul 2>&1

REM Verify files
echo.
echo Verifying files...
if exist "build\libs\duckdb.dll" (
    if exist "build\libs\duckdb.lib" (
        if exist "build\libs\duckdb.h" (
            if exist "build\libs\duckdb.hpp" (
                echo.
                echo ✅ All DuckDB library files downloaded successfully!
                echo.
                echo Files in build\libs\:
                dir /b build\libs\
                exit /b 0
            )
        )
    )
)

echo.
echo ❌ Some files are missing. Please check build\libs\
exit /b 1

